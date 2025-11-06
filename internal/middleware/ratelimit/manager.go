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

package ratelimit

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sjzar/ips/internal/middleware/ratelimit/model"
)

// Manager 限流管理器
type Manager struct {
	config          *model.RateLimitConfig
	globalLimiter   *GlobalLimiter
	ipLimiter       *IPLimiter
	endpointLimiter *EndpointLimiter
	metrics         *Metrics
}

// NewManager 创建限流管理器
func NewManager(cfg *model.RateLimitConfig) *Manager {
	if cfg == nil {
		cfg = model.DefaultRateLimitConfig()
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		log.Warnf("Invalid rate limit config: %v, using default", err)
		cfg = model.DefaultRateLimitConfig()
	}

	manager := &Manager{
		config: cfg,
	}

	// 初始化全局限流器
	if cfg.Enabled {
		manager.globalLimiter = NewGlobalLimiter(cfg.Global.Rate, cfg.Global.Burst)
		log.Infof("Global rate limiter enabled: %d req/s, burst %d", cfg.Global.Rate, cfg.Global.Burst)
	}

	// 初始化 IP 限流器
	if cfg.Enabled && cfg.IP.Enabled {
		// TTL 设置为 5 分钟，maxSize 设置为 10000
		manager.ipLimiter = NewIPLimiter(cfg.IP.Rate, cfg.IP.Burst, 5*time.Minute, 10000)
		log.Infof("IP rate limiter enabled: %d req/s, burst %d", cfg.IP.Rate, cfg.IP.Burst)

		// 记录白名单和黑名单
		if len(cfg.IP.Whitelist) > 0 {
			log.Infof("IP whitelist: %v", cfg.IP.Whitelist)
		}
		if len(cfg.IP.Blacklist) > 0 {
			log.Infof("IP blacklist: %v", cfg.IP.Blacklist)
		}
	}

	// 初始化接口级限流器
	if cfg.Enabled && len(cfg.Endpoints) > 0 {
		manager.endpointLimiter = NewEndpointLimiter(cfg.Endpoints)
		log.Infof("Endpoint rate limiter enabled: %d endpoints configured", len(cfg.Endpoints))
		for endpoint, epCfg := range cfg.Endpoints {
			log.Infof("  - %s: %d req/s, burst %d", endpoint, epCfg.Rate, epCfg.Burst)
		}
	}

	// 初始化监控指标
	if cfg.Enabled {
		manager.metrics = NewMetrics()
		log.Info("Rate limit metrics enabled")
	}

	return manager
}

// CheckGlobalLimit 检查全局限流
func (m *Manager) CheckGlobalLimit() bool {
	if !m.config.Enabled || m.globalLimiter == nil {
		return true
	}
	return m.globalLimiter.Allow()
}

// CheckIPLimit 检查 IP 限流
func (m *Manager) CheckIPLimit(ip string) bool {
	if !m.config.Enabled || !m.config.IP.Enabled || m.ipLimiter == nil {
		return true
	}
	return m.ipLimiter.Allow(ip)
}

// IsWhitelisted 检查 IP 是否在白名单中
func (m *Manager) IsWhitelisted(ip string) bool {
	if !m.config.Enabled || !m.config.IP.Enabled {
		return false
	}
	return m.config.IP.IsWhitelisted(ip)
}

// IsBlacklisted 检查 IP 是否在黑名单中
func (m *Manager) IsBlacklisted(ip string) bool {
	if !m.config.Enabled || !m.config.IP.Enabled {
		return false
	}
	return m.config.IP.IsBlacklisted(ip)
}

// GetConfig 获取配置
func (m *Manager) GetConfig() *model.RateLimitConfig {
	return m.config
}

// GetIPLimiterCount 获取当前 IP 限流器数量
func (m *Manager) GetIPLimiterCount() int {
	if m.ipLimiter == nil {
		return 0
	}
	return m.ipLimiter.Count()
}

// GetGlobalTokens 获取全局限流器当前令牌数
func (m *Manager) GetGlobalTokens() float64 {
	if m.globalLimiter == nil {
		return 0
	}
	return m.globalLimiter.Tokens()
}

// GetIPTokens 获取 IP 限流器当前令牌数
func (m *Manager) GetIPTokens(ip string) float64 {
	if m.ipLimiter == nil {
		return 0
	}
	return m.ipLimiter.Tokens(ip)
}

// CheckEndpointLimit 检查接口级限流
// 返回：是否允许通过, 速率限制, 突发容量
func (m *Manager) CheckEndpointLimit(path string) (allowed bool, limit float64, burst int) {
	if !m.config.Enabled || m.endpointLimiter == nil {
		return true, 0, 0
	}
	return m.endpointLimiter.CheckLimit(path)
}

// GetEndpointTokens 获取接口限流器当前令牌数
func (m *Manager) GetEndpointTokens(path string) float64 {
	if m.endpointLimiter == nil {
		return -1
	}
	return m.endpointLimiter.Tokens(path)
}

// GetEndpointLimiterCount 获取配置的接口限流数量
func (m *Manager) GetEndpointLimiterCount() int {
	if m.endpointLimiter == nil {
		return 0
	}
	return m.endpointLimiter.Count()
}

// HasEndpointLimit 检查是否为指定路径配置了限流
func (m *Manager) HasEndpointLimit(path string) bool {
	if m.endpointLimiter == nil {
		return false
	}
	return m.endpointLimiter.HasLimit(path)
}

// GetMetrics 获取监控指标
func (m *Manager) GetMetrics() *Metrics {
	return m.metrics
}

// UpdateIPLimitersMetric 更新 IP 限流器数量指标
func (m *Manager) UpdateIPLimitersMetric() {
	if m.metrics != nil && m.ipLimiter != nil {
		count := m.ipLimiter.Count()
		m.metrics.SetIPLimitersCount(count)
	}
}
