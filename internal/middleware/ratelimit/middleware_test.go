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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sjzar/ips/pkg/model"
)

func TestMiddleware_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create manager with disabled config
	cfg := &model.RateLimitConfig{
		Enabled: false,
	}
	manager := NewManager(cfg)

	// Setup router
	router := gin.New()
	router.Use(manager.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Make many requests - all should succeed
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d failed with code %d (rate limit should be disabled)", i+1, w.Code)
		}
	}
}

func TestMiddleware_IPRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create manager with IP rate limit
	cfg := &model.RateLimitConfig{
		Enabled: true,
		Global: model.GlobalLimitConfig{
			Rate:  1000,
			Burst: 2000,
		},
		IP: model.IPLimitConfig{
			Enabled:   true,
			Rate:      5,
			Burst:     10,
			Whitelist: []string{},
			Blacklist: []string{},
		},
		Response: model.ResponseConfig{
			StatusCode: 429,
			Message:    "Too Many Requests",
			RetryAfter: true,
		},
	}
	manager := NewManager(cfg)

	// Setup router
	router := gin.New()
	router.Use(manager.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Make requests within burst
	successCount := 0
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "1.2.3.4:12345"
		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful requests, got %d", successCount)
	}

	// Next request should be rate limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", w.Code)
	}

	// Check response headers
	if w.Header().Get("X-Ratelimit-Limit") == "" {
		t.Error("Missing X-Ratelimit-Limit header")
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("Missing Retry-After header")
	}
}

func TestMiddleware_Whitelist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create manager with whitelist
	cfg := &model.RateLimitConfig{
		Enabled: true,
		Global: model.GlobalLimitConfig{
			Rate:  1000,
			Burst: 2000,
		},
		IP: model.IPLimitConfig{
			Enabled:   true,
			Rate:      1,
			Burst:     2,
			Whitelist: []string{"127.0.0.1", "192.168.0.0/16"},
			Blacklist: []string{},
		},
	}
	manager := NewManager(cfg)

	// Setup router
	router := gin.New()
	router.Use(manager.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Whitelisted IP should not be rate limited
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Whitelisted IP request %d should succeed, got %d", i+1, w.Code)
		}
	}
}

func TestMiddleware_Blacklist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create manager with blacklist
	cfg := &model.RateLimitConfig{
		Enabled: true,
		Global: model.GlobalLimitConfig{
			Rate:  1000,
			Burst: 2000,
		},
		IP: model.IPLimitConfig{
			Enabled:   true,
			Rate:      1000,
			Burst:     2000,
			Whitelist: []string{},
			Blacklist: []string{"1.2.3.4", "10.0.0.0/8"},
		},
		Response: model.ResponseConfig{
			StatusCode: 429,
			Message:    "IP in blacklist",
		},
	}
	manager := NewManager(cfg)

	// Setup router
	router := gin.New()
	router.Use(manager.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Blacklisted IP should be rejected
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Blacklisted IP should be rejected, got %d", w.Code)
	}

	// Blacklisted network should be rejected
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.10.10.10:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Blacklisted network IP should be rejected, got %d", w.Code)
	}
}
