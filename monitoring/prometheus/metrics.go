// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package prometheus provides a Prometheus-based implementation of the
// MetricFactory abstraction.
package prometheus

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/google/trillian/monitoring"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// MetricFactory allows the creation of Prometheus-based metrics.
type MetricFactory struct {
	Prefix string
}

// NewCounter creates a new Counter object backed by Prometheus.
func (pmf MetricFactory) NewCounter(name, help string, labelNames []string) monitoring.Counter {
	if labelNames == nil || len(labelNames) == 0 {
		counter := prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: pmf.Prefix + name,
				Help: help,
			})
		prometheus.MustRegister(counter)
		return &Counter{single: counter}
	}

	vec := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: pmf.Prefix + name,
			Help: help,
		},
		labelNames)
	prometheus.MustRegister(vec)
	return &Counter{labelNames: labelNames, vec: vec}
}

// NewGauge creates a new Gauge object backed by Prometheus.
func (pmf MetricFactory) NewGauge(name, help string, labelNames []string) monitoring.Gauge {
	if labelNames == nil || len(labelNames) == 0 {
		gauge := prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: pmf.Prefix + name,
				Help: help,
			})
		prometheus.MustRegister(gauge)
		return &Gauge{single: gauge}
	}
	vec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: pmf.Prefix + name,
			Help: help,
		},
		labelNames)
	prometheus.MustRegister(vec)
	return &Gauge{labelNames: labelNames, vec: vec}
}

// NewHistogram creates a new Histogram object backed by Prometheus.
func (pmf MetricFactory) NewHistogram(name, help string, labelNames []string) monitoring.Histogram {
	if labelNames == nil || len(labelNames) == 0 {
		histogram := prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name: pmf.Prefix + name,
				Help: help,
			})
		prometheus.MustRegister(histogram)
		return &Histogram{single: histogram}
	}
	vec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: pmf.Prefix + name,
			Help: help,
		},
		labelNames)
	prometheus.MustRegister(vec)
	return &Histogram{labelNames: labelNames, vec: vec}
}

// Counter is a wrapper around a Prometheus Counter or CounterVec object.
type Counter struct {
	labelNames []string
	single     prometheus.Counter
	vec        *prometheus.CounterVec
}

// Inc adds 1 to a counter.
func (m *Counter) Inc(labelVals ...string) {
	if m.vec != nil {
		m.vec.With(labelsFor(m.labelNames, labelVals)).Inc()
	} else {
		m.single.Inc()
	}
}

// Add adds the given amount to a counter.
func (m *Counter) Add(val float64, labelVals ...string) {
	if m.vec != nil {
		m.vec.With(labelsFor(m.labelNames, labelVals)).Add(val)
	} else {
		m.single.Add(val)
	}
}

// Value returns the current amount of a counter.
func (m *Counter) Value(labelVals ...string) float64 {
	var metric prometheus.Metric
	if m.vec != nil {
		metric = m.vec.With(labelsFor(m.labelNames, labelVals))
	} else {
		metric = m.single
	}
	var metricpb dto.Metric
	if err := metric.Write(&metricpb); err != nil {
		glog.Errorf("failed to Write metric: %v", err)
		return 0.0
	}
	if metricpb.Counter == nil {
		glog.Errorf("counter field missing")
		return 0.0
	}
	return metricpb.Counter.GetValue()
}

// Gauge is a wrapper around a Prometheus Gauge or GaugeVec object.
type Gauge struct {
	labelNames []string
	single     prometheus.Gauge
	vec        *prometheus.GaugeVec
}

// Inc adds 1 to a gauge.
func (m *Gauge) Inc(labelVals ...string) {
	if m.vec != nil {
		m.vec.With(labelsFor(m.labelNames, labelVals)).Inc()
	} else {
		m.single.Inc()
	}
}

// Dec subtracts 1 from a gauge.
func (m *Gauge) Dec(labelVals ...string) {
	if m.vec != nil {
		m.vec.With(labelsFor(m.labelNames, labelVals)).Dec()
	} else {
		m.single.Dec()
	}
}

// Add adds given value to a gauge.
func (m *Gauge) Add(val float64, labelVals ...string) {
	if m.vec != nil {
		m.vec.With(labelsFor(m.labelNames, labelVals)).Add(val)
	} else {
		m.single.Add(val)
	}
}

// Set sets the value of a gauge.
func (m *Gauge) Set(val float64, labelVals ...string) {
	if m.vec != nil {
		m.vec.With(labelsFor(m.labelNames, labelVals)).Set(val)
	} else {
		m.single.Set(val)
	}
}

// Value returns the current amount of a gauge.
func (m *Gauge) Value(labelVals ...string) float64 {
	var metric prometheus.Metric
	if m.vec != nil {
		metric = m.vec.With(labelsFor(m.labelNames, labelVals))
	} else {
		metric = m.single
	}
	var metricpb dto.Metric
	if err := metric.Write(&metricpb); err != nil {
		glog.Errorf("failed to Write metric: %v", err)
		return 0.0
	}
	if metricpb.Gauge == nil {
		glog.Errorf("gauge field missing")
		return 0.0
	}
	return metricpb.Gauge.GetValue()
}

// Histogram is a wrapper around a Prometheus Histogram or HistogramVec object.
type Histogram struct {
	labelNames []string
	single     prometheus.Histogram
	vec        *prometheus.HistogramVec
}

// Observe adds a single observation to the histogram.
func (m *Histogram) Observe(val float64, labelVals ...string) {
	if m.vec != nil {
		m.vec.With(labelsFor(m.labelNames, labelVals)).Observe(val)
	} else {
		m.single.Observe(val)
	}
}

// Info returns the count and sum of observations for the histogram.
func (m *Histogram) Info(labelVals ...string) (uint64, float64) {
	var metric prometheus.Metric
	if m.vec != nil {
		metric = m.vec.MetricVec.With(labelsFor(m.labelNames, labelVals)).(prometheus.Metric)
	} else {
		metric = m.single
	}
	var metricpb dto.Metric
	if err := metric.Write(&metricpb); err != nil {
		glog.Errorf("failed to Write metric: %v", err)
		return 0, 0.0
	}
	histVal := metricpb.GetHistogram()
	if histVal == nil {
		glog.Errorf("histogram field missing")
		return 0, 0.0
	}
	return histVal.GetSampleCount(), histVal.GetSampleSum()
}

func labelsFor(names, values []string) prometheus.Labels {
	if len(names) != len(values) {
		panic(fmt.Sprintf("got %d (%v) values for %d labels (%v)", len(values), values, len(names), names))
	}
	labels := make(prometheus.Labels)
	for i, name := range names {
		labels[name] = values[i]
	}
	return labels
}