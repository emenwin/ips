/*
 * Copyright (c) 2023 shenjunzheng@gmail.com
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	"fmt"
	"net"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled   bool              `json:"enabled" yaml:"enabled"`     // 是否启用限流
	Global    GlobalLimitConfig `json:"global" yaml:"global"`       // 全局限流配置
	IP        IPLimitConfig     `json:"ip" yaml:"ip"`               // IP 限流配置
	Endpoints EndpointsConfig   `json:"endpoints" yaml:"endpoints"` // 接口级限流配置
	Response  ResponseConfig    `json:"response" yaml:"response"`   // 响应配置
}

// GlobalLimitConfig 全局限流配置
type GlobalLimitConfig struct {
	Rate  int `json:"rate" yaml:"rate"`   // 每秒允许的请求数
	Burst int `json:"burst" yaml:"burst"` // 突发容量
}

// IPLimitConfig IP 限流配置
type IPLimitConfig struct {
	Enabled   bool     `json:"enabled" yaml:"enabled"`     // 是否启用 IP 限流
	Rate      int      `json:"rate" yaml:"rate"`           // 每秒允许的请求数
	Burst     int      `json:"burst" yaml:"burst"`         // 突发容量
	Whitelist []string `json:"whitelist" yaml:"whitelist"` // IP 白名单（支持 CIDR）
	Blacklist []string `json:"blacklist" yaml:"blacklist"` // IP 黑名单（支持 CIDR）
}

// EndpointsConfig 接口级限流配置
type EndpointsConfig map[string]EndpointLimitConfig

// EndpointLimitConfig 单个接口限流配置
type EndpointLimitConfig struct {
	Rate  int `json:"rate" yaml:"rate"`   // 每秒允许的请求数
	Burst int `json:"burst" yaml:"burst"` // 突发容量
}

// ResponseConfig 限流响应配置
type ResponseConfig struct {
	StatusCode int    `json:"status_code" yaml:"status_code"` // HTTP 状态码，默认 429
	Message    string `json:"message" yaml:"message"`         // 响应消息
	RetryAfter bool   `json:"retry_after" yaml:"retry_after"` // 是否包含 Retry-After 头
}

// Validate 验证配置
func (c *RateLimitConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	// 验证全局限流配置
	if err := c.Global.Validate(); err != nil {
		return fmt.Errorf("global config: %w", err)
	}

	// 验证 IP 限流配置
	if c.IP.Enabled {
		if err := c.IP.Validate(); err != nil {
			return fmt.Errorf("ip config: %w", err)
		}
	}

	// 验证接口限流配置
	if err := c.Endpoints.Validate(); err != nil {
		return fmt.Errorf("endpoints config: %w", err)
	}

	// 验证响应配置
	if err := c.Response.Validate(); err != nil {
		return fmt.Errorf("response config: %w", err)
	}

	return nil
}

// Validate 验证全局限流配置
func (c *GlobalLimitConfig) Validate() error {
	if c.Rate <= 0 {
		return fmt.Errorf("rate must be greater than 0, got %d", c.Rate)
	}
	if c.Burst < c.Rate {
		return fmt.Errorf("burst (%d) should be >= rate (%d)", c.Burst, c.Rate)
	}
	return nil
}

// Validate 验证 IP 限流配置
func (c *IPLimitConfig) Validate() error {
	if c.Rate <= 0 {
		return fmt.Errorf("rate must be greater than 0, got %d", c.Rate)
	}
	if c.Burst < c.Rate {
		return fmt.Errorf("burst (%d) should be >= rate (%d)", c.Burst, c.Rate)
	}

	// 验证白名单 IP/CIDR
	for _, ip := range c.Whitelist {
		if err := validateIPOrCIDR(ip); err != nil {
			return fmt.Errorf("invalid whitelist entry %q: %w", ip, err)
		}
	}

	// 验证黑名单 IP/CIDR
	for _, ip := range c.Blacklist {
		if err := validateIPOrCIDR(ip); err != nil {
			return fmt.Errorf("invalid blacklist entry %q: %w", ip, err)
		}
	}

	return nil
}

// Validate 验证接口限流配置
func (c EndpointsConfig) Validate() error {
	for endpoint, config := range c {
		if config.Rate <= 0 {
			return fmt.Errorf("endpoint %q: rate must be greater than 0, got %d", endpoint, config.Rate)
		}
		if config.Burst < config.Rate {
			return fmt.Errorf("endpoint %q: burst (%d) should be >= rate (%d)", endpoint, config.Burst, config.Rate)
		}
	}
	return nil
}

// Validate 验证响应配置
func (c *ResponseConfig) Validate() error {
	if c.StatusCode <= 0 {
		c.StatusCode = 429 // 默认值
	}
	if c.StatusCode < 100 || c.StatusCode >= 600 {
		return fmt.Errorf("invalid status code: %d", c.StatusCode)
	}
	return nil
}

// DefaultRateLimitConfig 返回默认限流配置
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled: false,
		Global: GlobalLimitConfig{
			Rate:  300,
			Burst: 600,
		},
		IP: IPLimitConfig{
			Enabled:   true,
			Rate:      100,
			Burst:     200,
			Whitelist: []string{"127.0.0.1", "::1"},
			Blacklist: []string{},
		},
		Endpoints: EndpointsConfig{},
		Response: ResponseConfig{
			StatusCode: 429,
			Message:    "Too Many Requests",
			RetryAfter: true,
		},
	}
}

// validateIPOrCIDR 验证 IP 地址或 CIDR 网段
func validateIPOrCIDR(s string) error {
	// 尝试解析为 IP
	if ip := net.ParseIP(s); ip != nil {
		return nil
	}

	// 尝试解析为 CIDR
	if _, _, err := net.ParseCIDR(s); err == nil {
		return nil
	}

	return fmt.Errorf("not a valid IP or CIDR: %s", s)
}

// IsWhitelisted 检查 IP 是否在白名单中
func (c *IPLimitConfig) IsWhitelisted(ip string) bool {
	return matchIPList(ip, c.Whitelist)
}

// IsBlacklisted 检查 IP 是否在黑名单中
func (c *IPLimitConfig) IsBlacklisted(ip string) bool {
	return matchIPList(ip, c.Blacklist)
}

// matchIPList 检查 IP 是否匹配列表中的任意项（支持 CIDR）
func matchIPList(ipStr string, list []string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for _, entry := range list {
		// 尝试作为单个 IP 匹配
		if entryIP := net.ParseIP(entry); entryIP != nil {
			if ip.Equal(entryIP) {
				return true
			}
			continue
		}

		// 尝试作为 CIDR 网段匹配
		if _, ipNet, err := net.ParseCIDR(entry); err == nil {
			if ipNet.Contains(ip) {
				return true
			}
		}
	}

	return false
}
