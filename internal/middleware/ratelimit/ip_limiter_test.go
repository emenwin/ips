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
	"time"
)

func TestIPLimiter(t *testing.T) {
	limiter := NewIPLimiter(10, 20, time.Minute, 100)

	// Test allowing requests from same IP
	for i := 0; i < 20; i++ {
		if !limiter.Allow("1.2.3.4") {
			t.Errorf("Request %d from 1.2.3.4 should be allowed", i+1)
		}
	}

	// Test rate limiting for same IP
	if limiter.Allow("1.2.3.4") {
		t.Error("Request 21 from 1.2.3.4 should be rejected")
	}

	// Test that different IP has separate limit
	if !limiter.Allow("5.6.7.8") {
		t.Error("First request from 5.6.7.8 should be allowed")
	}
}

func TestIPLimiter_Count(t *testing.T) {
	limiter := NewIPLimiter(10, 20, time.Minute, 100)

	// Initially should have 0 limiters
	if count := limiter.Count(); count != 0 {
		t.Errorf("Expected 0 limiters, got %d", count)
	}

	// After one request, should have 1 limiter
	limiter.Allow("1.2.3.4")
	if count := limiter.Count(); count != 1 {
		t.Errorf("Expected 1 limiter, got %d", count)
	}

	// After request from different IP, should have 2 limiters
	limiter.Allow("5.6.7.8")
	if count := limiter.Count(); count != 2 {
		t.Errorf("Expected 2 limiters, got %d", count)
	}
}

func TestIPLimiter_Tokens(t *testing.T) {
	limiter := NewIPLimiter(10, 20, time.Minute, 100)

	// Initial tokens for new IP
	tokens := limiter.Tokens("1.2.3.4")
	if tokens < 19 || tokens > 20 {
		t.Errorf("Expected ~20 initial tokens, got %f", tokens)
	}

	// Consume some tokens
	limiter.Allow("1.2.3.4")
	limiter.Allow("1.2.3.4")
	tokens = limiter.Tokens("1.2.3.4")
	if tokens < 17 || tokens > 19 {
		t.Errorf("Expected ~18 tokens after 2 requests, got %f", tokens)
	}
}

func TestIPLimiter_MaxSize(t *testing.T) {
	limiter := NewIPLimiter(10, 20, time.Minute, 3)

	// Add 3 IPs (at max)
	limiter.Allow("1.2.3.1")
	limiter.Allow("1.2.3.2")
	limiter.Allow("1.2.3.3")

	if count := limiter.Count(); count != 3 {
		t.Errorf("Expected 3 limiters, got %d", count)
	}

	// 4th IP should still work (might trigger cleanup or use temporary limiter)
	if !limiter.Allow("1.2.3.4") {
		t.Error("4th IP should be allowed (temporary limiter)")
	}
}

func TestIPLimiter_Clear(t *testing.T) {
	limiter := NewIPLimiter(10, 20, time.Minute, 100)

	// Add some limiters
	limiter.Allow("1.2.3.4")
	limiter.Allow("5.6.7.8")

	// Clear all
	limiter.Clear()

	if count := limiter.Count(); count != 0 {
		t.Errorf("Expected 0 limiters after clear, got %d", count)
	}
}
