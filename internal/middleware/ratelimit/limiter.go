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

// Limiter 限流器接口
type Limiter interface {
	// Allow 检查是否允许请求
	// 返回 true 表示允许，false 表示拒绝
	Allow() bool

	// AllowN 检查是否允许 n 个请求
	AllowN(n int) bool

	// Wait 等待直到允许请求（阻塞）
	// 返回 error 如果 context 被取消
	Wait(ctx context.Context) error

	// Tokens 返回当前可用的令牌数
	Tokens() float64

	// Limit 返回限流器的速率限制
	Limit() float64

	// Burst 返回限流器的突发容量
	Burst() int
}
