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
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Middleware 返回 Gin 限流中间件
func (m *Manager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果限流未启用，直接通过
		if !m.config.Enabled {
			c.Next()
			return
		}

		// 获取客户端 IP
		clientIP := c.ClientIP()
		requestPath := c.Request.URL.Path

		// 1. 检查黑名单
		if m.IsBlacklisted(clientIP) {
			m.rejectRequest(c, "IP in blacklist", clientIP, "blacklist")
			if m.metrics != nil {
				m.metrics.RecordRequest("blacklist", "rejected")
				m.metrics.RecordRejection("blacklist", requestPath)
			}
			return
		}

		// 2. 检查白名单（白名单中的 IP 跳过限流）
		if m.IsWhitelisted(clientIP) {
			if m.metrics != nil {
				m.metrics.RecordRequest("whitelist", "allowed")
			}
			c.Next()
			return
		}

		// 3. 全局限流检查
		if !m.CheckGlobalLimit() {
			m.rejectRequest(c, "Global rate limit exceeded", clientIP, "global")
			if m.metrics != nil {
				m.metrics.RecordRequest("global", "rejected")
				m.metrics.RecordRejection("global", requestPath)
			}
			return
		}
		if m.metrics != nil {
			m.metrics.RecordRequest("global", "allowed")
		}

		// 4. 接口级限流检查
		if m.endpointLimiter != nil {
			if allowed, _, _ := m.CheckEndpointLimit(requestPath); !allowed {
				m.rejectRequest(c, "Endpoint rate limit exceeded", clientIP, "endpoint")
				if m.metrics != nil {
					m.metrics.RecordRequest("endpoint", "rejected")
					m.metrics.RecordRejection("endpoint", requestPath)
				}
				return
			}
			if m.metrics != nil {
				m.metrics.RecordRequest("endpoint", "allowed")
			}
		}

		// 5. IP 限流检查
		if m.config.IP.Enabled && !m.CheckIPLimit(clientIP) {
			m.rejectRequest(c, "IP rate limit exceeded", clientIP, "ip")
			if m.metrics != nil {
				m.metrics.RecordRequest("ip", "rejected")
				m.metrics.RecordRejection("ip", requestPath)
			}
			return
		}
		if m.config.IP.Enabled && m.metrics != nil {
			m.metrics.RecordRequest("ip", "allowed")
		}

		// 6. 设置响应头
		m.setRateLimitHeaders(c, clientIP)

		// 定期更新 IP 限流器数量指标
		if m.metrics != nil {
			m.UpdateIPLimitersMetric()
		}

		// 请求通过
		c.Next()
	}
}

// rejectRequest 拒绝请求
func (m *Manager) rejectRequest(c *gin.Context, reason, ip, limitType string) {
	// 设置响应头
	m.setRateLimitHeaders(c, ip)

	// 设置 Retry-After 头
	if m.config.Response.RetryAfter {
		c.Header("Retry-After", "1")
	}

	// 记录日志
	log.WithFields(log.Fields{
		"ip":         ip,
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
		"reason":     reason,
		"limit_type": limitType,
	}).Warn("Rate limit exceeded")

	// 返回错误响应
	statusCode := m.config.Response.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusTooManyRequests
	}

	message := m.config.Response.Message
	if message == "" {
		message = reason
	}

	c.JSON(statusCode, gin.H{
		"error":   "Too Many Requests",
		"message": message,
	})

	c.Abort()
}

// setRateLimitHeaders 设置限流相关的响应头
func (m *Manager) setRateLimitHeaders(c *gin.Context, ip string) {
	requestPath := c.Request.URL.Path

	// X-RateLimit-Limit: 时间窗口内允许的最大请求数
	var limit int
	var remaining int

	// 优先级：接口级 > IP 级 > 全局
	if m.endpointLimiter != nil && m.HasEndpointLimit(requestPath) {
		// 接口级限流
		limit = int(m.endpointLimiter.Limit(requestPath))
		tokens := m.GetEndpointTokens(requestPath)
		remaining = int(tokens)
	} else if m.config.IP.Enabled && m.ipLimiter != nil {
		// IP 级限流
		limit = m.config.IP.Rate
		tokens := m.GetIPTokens(ip)
		remaining = int(tokens)
	} else if m.globalLimiter != nil {
		// 全局限流
		limit = m.config.Global.Rate
		tokens := m.GetGlobalTokens()
		remaining = int(tokens)
	}

	if remaining < 0 {
		remaining = 0
	}

	c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

	// X-RateLimit-Reset: 限流重置的 Unix 时间戳（当前时间 + 1秒）
	reset := time.Now().Add(time.Second).Unix()
	c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", reset))
}
