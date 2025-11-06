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
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricsInstance *Metrics
	metricsOnce     sync.Once
)

// Metrics 限流监控指标
type Metrics struct {
	requestsTotal   *prometheus.CounterVec
	rejectedTotal   *prometheus.CounterVec
	ipLimitersCount prometheus.Gauge
	durationSeconds *prometheus.HistogramVec
}

// NewMetrics 创建监控指标（使用单例模式）
func NewMetrics() *Metrics {
	metricsOnce.Do(func() {
		metricsInstance = &Metrics{
			// 请求总数（按限流类型和结果分类）
			requestsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "ips_ratelimit_requests_total",
					Help: "Total number of rate limit checks",
				},
				[]string{"limit_type", "result"},
			),

			// 拒绝总数（按限流类型分类）
			rejectedTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "ips_ratelimit_rejected_total",
					Help: "Total number of rate limited requests",
				},
				[]string{"limit_type", "path"},
			),

			// IP 限流器数量
			ipLimitersCount: promauto.NewGauge(
				prometheus.GaugeOpts{
					Name: "ips_ratelimit_ip_limiters_count",
					Help: "Number of active IP limiters",
				},
			),

			// 限流检查延迟
			durationSeconds: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "ips_ratelimit_duration_seconds",
					Help:    "Rate limit check duration in seconds",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"limit_type"},
			),
		}
	})
	return metricsInstance
}

// RecordRequest 记录请求（限流检查）
// limitType: "global", "ip", "endpoint"
// result: "allowed", "rejected"
func (m *Metrics) RecordRequest(limitType, result string) {
	m.requestsTotal.WithLabelValues(limitType, result).Inc()
}

// RecordRejection 记录拒绝
// limitType: "global", "ip", "endpoint", "blacklist"
// path: 请求路径
func (m *Metrics) RecordRejection(limitType, path string) {
	m.rejectedTotal.WithLabelValues(limitType, path).Inc()
}

// SetIPLimitersCount 设置 IP 限流器数量
func (m *Metrics) SetIPLimitersCount(count int) {
	m.ipLimitersCount.Set(float64(count))
}

// RecordDuration 记录限流检查延迟
// limitType: "global", "ip", "endpoint"
// duration: 延迟（秒）
func (m *Metrics) RecordDuration(limitType string, duration float64) {
	m.durationSeconds.WithLabelValues(limitType).Observe(duration)
}

// ObserveDuration 返回一个 Timer 用于记录延迟
func (m *Metrics) ObserveDuration(limitType string) prometheus.Observer {
	return m.durationSeconds.WithLabelValues(limitType)
}
