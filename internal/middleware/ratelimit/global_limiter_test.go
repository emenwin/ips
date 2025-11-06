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

func TestGlobalLimiter(t *testing.T) {
	limiter := NewGlobalLimiter(10, 20)

	// Test initial state
	if limiter.Limit() != 10 {
		t.Errorf("Expected limit 10, got %f", limiter.Limit())
	}
	if limiter.Burst() != 20 {
		t.Errorf("Expected burst 20, got %d", limiter.Burst())
	}

	// Test burst capacity
	for i := 0; i < 20; i++ {
		if !limiter.Allow() {
			t.Errorf("Request %d should be allowed (within burst)", i+1)
		}
	}

	// Test rate limiting
	if limiter.Allow() {
		t.Error("Request 21 should be rejected (exceeds burst)")
	}

	// Wait for token refill
	time.Sleep(150 * time.Millisecond)
	if !limiter.Allow() {
		t.Error("Request should be allowed after waiting for refill")
	}
}

func TestGlobalLimiter_AllowN(t *testing.T) {
	limiter := NewGlobalLimiter(10, 20)

	// Test allowing multiple requests at once
	if !limiter.AllowN(10) {
		t.Error("AllowN(10) should succeed")
	}

	// Test exceeding burst
	if limiter.AllowN(15) {
		t.Error("AllowN(15) should fail (only 10 tokens left)")
	}
}

func TestGlobalLimiter_Tokens(t *testing.T) {
	limiter := NewGlobalLimiter(10, 20)

	// Initial tokens should be burst capacity
	tokens := limiter.Tokens()
	if tokens < 19 || tokens > 20 {
		t.Errorf("Expected ~20 initial tokens, got %f", tokens)
	}

	// Consume some tokens
	limiter.Allow()
	limiter.Allow()
	tokens = limiter.Tokens()
	if tokens < 17 || tokens > 19 {
		t.Errorf("Expected ~18 tokens after 2 requests, got %f", tokens)
	}
}
