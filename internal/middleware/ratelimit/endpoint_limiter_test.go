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
	"testing"

	"github.com/sjzar/ips/internal/middleware/ratelimit/model"
)

func TestEndpointLimiter_ExactMatch(t *testing.T) {
	config := model.EndpointsConfig{
		"/api/v1/query": {Rate: 10, Burst: 20},
		"/api/v1/ip":    {Rate: 5, Burst: 10},
	}

	limiter := NewEndpointLimiter(config)

	// 测试精确匹配
	allowed, limit, burst := limiter.CheckLimit("/api/v1/query")
	if !allowed {
		t.Error("Expected first request to be allowed")
	}
	if limit != 10 {
		t.Errorf("Expected limit 10, got %f", limit)
	}
	if burst != 20 {
		t.Errorf("Expected burst 20, got %d", burst)
	}

	// 测试不同接口
	allowed, limit, burst = limiter.CheckLimit("/api/v1/ip")
	if !allowed {
		t.Error("Expected first request to /api/v1/ip to be allowed")
	}
	if limit != 5 {
		t.Errorf("Expected limit 5, got %f", limit)
	}
	if burst != 10 {
		t.Errorf("Expected burst 10, got %d", burst)
	}
}

func TestEndpointLimiter_PrefixMatch(t *testing.T) {
	config := model.EndpointsConfig{
		"/api/v1": {Rate: 20, Burst: 40},
	}

	limiter := NewEndpointLimiter(config)

	// 测试前缀匹配
	testCases := []string{
		"/api/v1/query",
		"/api/v1/ip",
		"/api/v1/search?q=test",
	}

	for _, path := range testCases {
		allowed, limit, _ := limiter.CheckLimit(path)
		if !allowed {
			t.Errorf("Expected request to %s to be allowed", path)
		}
		if limit != 20 {
			t.Errorf("For path %s, expected limit 20, got %f", path, limit)
		}
	}

	// 不应该匹配的路径
	allowed, limit, _ := limiter.CheckLimit("/api/v10/query")
	if !allowed {
		t.Error("Expected request to /api/v10/query to be allowed (no limit)")
	}
	if limit != 0 {
		t.Errorf("Expected limit 0 (no limit), got %f", limit)
	}
}

func TestEndpointLimiter_LongestMatch(t *testing.T) {
	config := model.EndpointsConfig{
		"/api":          {Rate: 100, Burst: 200},
		"/api/v1":       {Rate: 50, Burst: 100},
		"/api/v1/query": {Rate: 10, Burst: 20},
	}

	limiter := NewEndpointLimiter(config)

	// 应该匹配最具体的路径
	_, limit, _ := limiter.CheckLimit("/api/v1/query")
	if limit != 10 {
		t.Errorf("Expected most specific limit 10, got %f", limit)
	}

	_, limit, _ = limiter.CheckLimit("/api/v1/ip")
	if limit != 50 {
		t.Errorf("Expected second-level limit 50, got %f", limit)
	}

	_, limit, _ = limiter.CheckLimit("/api/v2/query")
	if limit != 100 {
		t.Errorf("Expected top-level limit 100, got %f", limit)
	}
}

func TestEndpointLimiter_RateLimit(t *testing.T) {
	config := model.EndpointsConfig{
		"/api/test": {Rate: 2, Burst: 3},
	}

	limiter := NewEndpointLimiter(config)

	// 前 3 个请求应该通过（burst=3）
	for i := 0; i < 3; i++ {
		allowed, _, _ := limiter.CheckLimit("/api/test")
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 第 4 个请求应该被限流
	allowed, _, _ := limiter.CheckLimit("/api/test")
	if allowed {
		t.Error("Request 4 should be rate limited")
	}
}

func TestEndpointLimiter_NoLimit(t *testing.T) {
	config := model.EndpointsConfig{
		"/api/v1/query": {Rate: 10, Burst: 20},
	}

	limiter := NewEndpointLimiter(config)

	// 未配置限流的路径应该允许通过
	allowed, limit, burst := limiter.CheckLimit("/api/v2/query")
	if !allowed {
		t.Error("Expected request to unconfigured path to be allowed")
	}
	if limit != 0 {
		t.Errorf("Expected limit 0 for unconfigured path, got %f", limit)
	}
	if burst != 0 {
		t.Errorf("Expected burst 0 for unconfigured path, got %d", burst)
	}
}

func TestEndpointLimiter_Tokens(t *testing.T) {
	config := model.EndpointsConfig{
		"/api/test": {Rate: 10, Burst: 20},
	}

	limiter := NewEndpointLimiter(config)

	// 初始应该有 20 个令牌
	tokens := limiter.Tokens("/api/test")
	if tokens < 19.9 || tokens > 20.1 {
		t.Errorf("Expected ~20 tokens initially, got %f", tokens)
	}

	// 消耗一个令牌
	limiter.CheckLimit("/api/test")
	tokens = limiter.Tokens("/api/test")
	if tokens < 18.9 || tokens > 19.1 {
		t.Errorf("Expected ~19 tokens after one request, got %f", tokens)
	}
}

func TestEndpointLimiter_HasLimit(t *testing.T) {
	config := model.EndpointsConfig{
		"/api/v1": {Rate: 10, Burst: 20},
	}

	limiter := NewEndpointLimiter(config)

	// 有限流配置的路径
	if !limiter.HasLimit("/api/v1/query") {
		t.Error("Expected /api/v1/query to have limit (prefix match)")
	}

	// 没有限流配置的路径
	if limiter.HasLimit("/api/v2/query") {
		t.Error("Expected /api/v2/query to have no limit")
	}
}

func TestEndpointLimiter_Count(t *testing.T) {
	config := model.EndpointsConfig{
		"/api/v1/query":  {Rate: 10, Burst: 20},
		"/api/v1/ip":     {Rate: 5, Burst: 10},
		"/api/v1/search": {Rate: 20, Burst: 40},
	}

	limiter := NewEndpointLimiter(config)

	count := limiter.Count()
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

func TestEndpointLimiter_PathClean(t *testing.T) {
	config := model.EndpointsConfig{
		"/api/v1/query": {Rate: 10, Burst: 20},
	}

	limiter := NewEndpointLimiter(config)

	// 测试路径清理
	testCases := []string{
		"/api/v1/query",
		"/api/v1/query/",
		"/api/v1//query",
		"/api/v1/./query",
	}

	for _, path := range testCases {
		_, limit, _ := limiter.CheckLimit(path)
		if limit != 10 {
			t.Errorf("Path %s should match, expected limit 10, got %f", path, limit)
		}
	}
}
