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
)

// GlobalLimiter 全局限流器
type GlobalLimiter struct {
	limiter Limiter
}

// NewGlobalLimiter 创建全局限流器
// rate: 每秒允许的请求数
// burst: 突发容量
func NewGlobalLimiter(r int, b int) *GlobalLimiter {
	return &GlobalLimiter{
		limiter: NewTokenLimiter(float64(r), b),
	}
}

// Allow 检查是否允许请求
func (g *GlobalLimiter) Allow() bool {
	return g.limiter.Allow()
}

// AllowN 检查是否允许 n 个请求
func (g *GlobalLimiter) AllowN(n int) bool {
	return g.limiter.AllowN(n)
}

// Wait 等待直到允许请求
func (g *GlobalLimiter) Wait(ctx context.Context) error {
	return g.limiter.Wait(ctx)
}

// Tokens 返回当前可用的令牌数
func (g *GlobalLimiter) Tokens() float64 {
	return g.limiter.Tokens()
}

// Limit 返回限流器的速率限制
func (g *GlobalLimiter) Limit() float64 {
	return g.limiter.Limit()
}

// Burst 返回限流器的突发容量
func (g *GlobalLimiter) Burst() int {
	return g.limiter.Burst()
}
