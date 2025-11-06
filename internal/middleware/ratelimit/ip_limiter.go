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
	"context"
	"sync"
	"time"
)

// ipLimiterEntry IP 限流器条目
type ipLimiterEntry struct {
	limiter  Limiter
	lastSeen time.Time
}

// IPLimiter IP 限流器
// 为每个 IP 地址维护独立的限流器
type IPLimiter struct {
	limiters sync.Map // map[string]*ipLimiterEntry
	rate     float64
	burst    int
	ttl      time.Duration
	maxSize  int
	mu       sync.RWMutex
	count    int
}

// NewIPLimiter 创建 IP 限流器
// r: 每秒允许的请求数
// b: 突发容量
// ttl: 限流器过期时间
// maxSize: 最大限流器数量（0 表示不限制）
func NewIPLimiter(r int, b int, ttl time.Duration, maxSize int) *IPLimiter {
	limiter := &IPLimiter{
		rate:    float64(r),
		burst:   b,
		ttl:     ttl,
		maxSize: maxSize,
		count:   0,
	}

	// 启动清理协程
	go limiter.cleanup()

	return limiter
}

// GetLimiter 获取或创建 IP 的限流器
func (l *IPLimiter) GetLimiter(ip string) Limiter {
	// 快速路径：尝试获取现有限流器
	if entry, exists := l.limiters.Load(ip); exists {
		e := entry.(*ipLimiterEntry)
		e.lastSeen = time.Now()
		return e.limiter
	}

	// 慢路径：创建新限流器
	l.mu.Lock()
	defer l.mu.Unlock()

	// 双重检查
	if entry, exists := l.limiters.Load(ip); exists {
		e := entry.(*ipLimiterEntry)
		e.lastSeen = time.Now()
		return e.limiter
	}

	// 检查是否超过最大限制
	if l.maxSize > 0 && l.count >= l.maxSize {
		// 如果达到上限，清理一些过期的
		l.cleanupExpired()
		// 如果清理后仍然超限，返回一个临时限流器
		if l.count >= l.maxSize {
			return NewTokenLimiter(l.rate, l.burst)
		}
	}

	// 创建新限流器
	newLimiter := NewTokenLimiter(l.rate, l.burst)
	entry := &ipLimiterEntry{
		limiter:  newLimiter,
		lastSeen: time.Now(),
	}
	l.limiters.Store(ip, entry)
	l.count++

	return newLimiter
}

// Allow 检查 IP 是否允许请求
func (l *IPLimiter) Allow(ip string) bool {
	limiter := l.GetLimiter(ip)
	return limiter.Allow()
}

// AllowN 检查 IP 是否允许 n 个请求
func (l *IPLimiter) AllowN(ip string, n int) bool {
	limiter := l.GetLimiter(ip)
	return limiter.AllowN(n)
}

// Wait 等待直到允许 IP 的请求
func (l *IPLimiter) Wait(ctx context.Context, ip string) error {
	limiter := l.GetLimiter(ip)
	return limiter.Wait(ctx)
}

// Tokens 返回 IP 当前可用的令牌数
func (l *IPLimiter) Tokens(ip string) float64 {
	limiter := l.GetLimiter(ip)
	return limiter.Tokens()
}

// Limit 返回限流器的速率限制
func (l *IPLimiter) Limit() float64 {
	return l.rate
}

// Burst 返回限流器的突发容量
func (l *IPLimiter) Burst() int {
	return l.burst
}

// Count 返回当前活跃的限流器数量
func (l *IPLimiter) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.count
}

// cleanup 定期清理过期的限流器
func (l *IPLimiter) cleanup() {
	if l.ttl <= 0 {
		return
	}

	ticker := time.NewTicker(l.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanupExpired()
	}
}

// cleanupExpired 清理过期的限流器
func (l *IPLimiter) cleanupExpired() {
	now := time.Now()
	toDelete := make([]string, 0)

	l.limiters.Range(func(key, value interface{}) bool {
		ip := key.(string)
		entry := value.(*ipLimiterEntry)

		// 如果令牌桶已满且超过 TTL，则删除
		if entry.limiter.Tokens() >= float64(l.burst) && now.Sub(entry.lastSeen) > l.ttl {
			toDelete = append(toDelete, ip)
		}
		return true
	})

	// 执行删除
	if len(toDelete) > 0 {
		l.mu.Lock()
		for _, ip := range toDelete {
			l.limiters.Delete(ip)
			l.count--
		}
		l.mu.Unlock()
	}
}

// Clear 清除所有限流器
func (l *IPLimiter) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.limiters.Range(func(key, value interface{}) bool {
		l.limiters.Delete(key)
		return true
	})
	l.count = 0
}
