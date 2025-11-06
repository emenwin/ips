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
	"path"
	"sync"

	"github.com/sjzar/ips/internal/middleware/ratelimit/model"
)

// EndpointLimiter 接口级限流器
type EndpointLimiter struct {
	limiters map[string]Limiter
	mu       sync.RWMutex
}

// NewEndpointLimiter 创建接口限流器
func NewEndpointLimiter(config model.EndpointsConfig) *EndpointLimiter {
	limiters := make(map[string]Limiter, len(config))

	for endpoint, cfg := range config {
		limiters[endpoint] = NewTokenLimiter(float64(cfg.Rate), cfg.Burst)
	}

	return &EndpointLimiter{
		limiters: limiters,
	}
}

// CheckLimit 检查指定路径的限流
// 返回：是否允许通过, 当前限流器的速率, 突发容量
func (e *EndpointLimiter) CheckLimit(requestPath string) (allowed bool, limit float64, burst int) {
	limiter := e.getLimiterForPath(requestPath)
	if limiter == nil {
		// 没有配置该路径的限流，允许通过
		return true, 0, 0
	}

	allowed = limiter.Allow()
	limit = float64(limiter.Limit())
	burst = limiter.Burst()
	return
}

// getLimiterForPath 获取路径对应的限流器
// 支持精确匹配和前缀匹配
func (e *EndpointLimiter) getLimiterForPath(requestPath string) Limiter {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 1. 精确匹配
	if limiter, ok := e.limiters[requestPath]; ok {
		return limiter
	}

	// 2. 清理路径并再次尝试精确匹配
	cleanPath := path.Clean(requestPath)
	if limiter, ok := e.limiters[cleanPath]; ok {
		return limiter
	}

	// 3. 前缀匹配（找到最长匹配）
	var bestMatch string
	var bestLimiter Limiter

	for endpoint, limiter := range e.limiters {
		// 检查是否为路径前缀
		if len(endpoint) > len(bestMatch) && isPathPrefix(endpoint, cleanPath) {
			bestMatch = endpoint
			bestLimiter = limiter
		}
	}

	return bestLimiter
}

// isPathPrefix 检查 prefix 是否是 path 的前缀
// 例如："/api/v1" 是 "/api/v1/query" 的前缀
func isPathPrefix(prefix, path string) bool {
	if len(prefix) > len(path) {
		return false
	}

	// 精确匹配
	if prefix == path {
		return true
	}

	// 前缀匹配，确保边界正确
	// "/api/v1" 应该匹配 "/api/v1/query"
	// 但不应该匹配 "/api/v10/query"
	if path[:len(prefix)] == prefix {
		// 检查下一个字符是否为 '/'
		if len(path) > len(prefix) && path[len(prefix)] == '/' {
			return true
		}
		// 或者 prefix 本身就以 '/' 结尾
		if len(prefix) > 0 && prefix[len(prefix)-1] == '/' {
			return true
		}
	}

	return false
}

// Tokens 返回指定路径的可用令牌数
func (e *EndpointLimiter) Tokens(requestPath string) float64 {
	limiter := e.getLimiterForPath(requestPath)
	if limiter == nil {
		return -1 // 表示没有配置限流
	}
	return limiter.Tokens()
}

// Limit 返回指定路径的速率限制
func (e *EndpointLimiter) Limit(requestPath string) float64 {
	limiter := e.getLimiterForPath(requestPath)
	if limiter == nil {
		return 0
	}
	return limiter.Limit()
}

// Burst 返回指定路径的突发容量
func (e *EndpointLimiter) Burst(requestPath string) int {
	limiter := e.getLimiterForPath(requestPath)
	if limiter == nil {
		return 0
	}
	return limiter.Burst()
}

// HasLimit 检查是否为指定路径配置了限流
func (e *EndpointLimiter) HasLimit(requestPath string) bool {
	return e.getLimiterForPath(requestPath) != nil
}

// Count 返回配置的接口限流数量
func (e *EndpointLimiter) Count() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.limiters)
}
