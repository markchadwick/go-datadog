// Datadog reporter for the [go-metrics](https://github.com/rcrowley/go-metrics)
// library.
package datadog

import (
	"github.com/rcrowley/go-metrics"
	"log"
	"regexp"
	"strings"
	"time"
)

type MetricsReporter struct {
	client   *Client
	registry metrics.Registry
}

// Expect the tags in the pattern
// namespace.metricName[tag1:value1,tag2:value2,etc....]
var tagPattern = regexp.MustCompile("([\\w\\.]+)\\[([\\w\\W]+)\\]")

// Create an un-started MetricsReporter. In most circumstances, the
// `metrics.DefaultRegistry` will suffice for the required `metrics.Registry`.
// The recreated `MetricsReporter` will not be started. Invoke `go r.Start(..)` with
// a `time.Duration` to enable reporting.
func Reporter(c *Client, r metrics.Registry) *MetricsReporter {
	return &MetricsReporter{
		client:   c,
		registry: r,
	}
}

// Start this reporter in a blocking fashion, pushing series data to datadog at
// the specified interval. If any errors occur, they will be logged to the
// default logger, and further updates will continue.
//
// Scheduling is done with a `time.Ticker`, so non-overlapping intervals are
// absolute, not based on the finish time of the previous event. They are,
// however, serial.
func (mr *MetricsReporter) Start(d time.Duration) {
	ticker := time.NewTicker(d)
	for _ = range ticker.C {
		if err := mr.Report(); err != nil {
			log.Printf("Datadog series error: %s", err.Error())
		}
	}
}

// POST a single series report to the Datadog API. A 200 or 202 is expected for
// this to complete without error.
func (mr *MetricsReporter) Report() error {
	return mr.client.PostSeries(mr.Series())
}

// For each metric assocaited with the current Registry, convert it to a
// `Series` message, and return them all as a single array. The series messages
// will have the current hostname of the `Client`.
func (mr *MetricsReporter) Series() []*Series {
	now := time.Now().Unix()
	series := make([]*Series, 0)
	mr.registry.Each(func(name string, metric interface{}) {
		series = append(series, mr.series(now, name, metric)...)
	})
	return series
}

// Switch through the known types of meters delegating out to specific methods.
// If an unknown metric is encountered, this will return nil.
func (mr *MetricsReporter) series(t int64, name string, i interface{}) []*Series {
	switch m := i.(type) {
	case metrics.Counter:
		return mr.counterSeries(t, name, m)
	case metrics.Gauge:
		return mr.gaugeSeries(t, name, m)
	case metrics.Healthcheck:
		// TODO: Not implemented
	case metrics.Histogram:
		return mr.histogramSeries(t, name, m)
	case metrics.Meter:
		return mr.meterSeries(t, name, m)
	case metrics.Timer:
		return mr.timerSeries(t, name, m)
	}
	return nil
}

func (mr *MetricsReporter) counterSeries(t int64, id string,
	counter metrics.Counter) []*Series {
	name, tags := splitNameAndTags(id)
	counter.Inc(0)
	return []*Series{
		mr.counterI(name+".count", t, counter.Count(), tags),
	}
}

func (mr *MetricsReporter) gaugeSeries(t int64, id string,
	gauge metrics.Gauge) []*Series {
	name, tags := splitNameAndTags(id)
	return []*Series{
		mr.gaugeI(name+".value", t, gauge.Value(), tags),
	}
}

func (mr *MetricsReporter) histogramSeries(t int64, id string,
	h metrics.Histogram) []*Series {
	ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
	name, tags := splitNameAndTags(id)

	return []*Series{
		mr.counterI(name+".count", t, h.Count(), tags),
		mr.counterI(name+".min", t, h.Min(), tags),
		mr.counterI(name+".max", t, h.Max(), tags),
		mr.counterF(name+".mean", t, h.Mean(), tags),
		mr.counterF(name+".stddev", t, h.StdDev(), tags),
		mr.counterF(name+".median", t, ps[0], tags),
		mr.counterF(name+".percentile.75", t, ps[1], tags),
		mr.counterF(name+".percentile.95", t, ps[2], tags),
		mr.counterF(name+".percentile.99", t, ps[3], tags),
		mr.counterF(name+".percentile.999", t, ps[4], tags),
	}
}

func (mr *MetricsReporter) meterSeries(t int64, id string,
	m metrics.Meter) []*Series {
	name, tags := splitNameAndTags(id)
	m.Mark(0)
	return []*Series{
		mr.counterI(name+".count", t, m.Count(), tags),
		mr.counterF(name+".rate.1min", t, m.Rate1(), tags),
		mr.counterF(name+".rate.5min", t, m.Rate5(), tags),
		mr.counterF(name+".rate.15min", t, m.Rate15(), tags),
		mr.counterF(name+".rate.mean", t, m.RateMean(), tags),
	}
}

func (mr *MetricsReporter) timerSeries(t int64, id string,
	m metrics.Timer) []*Series {
	ps := m.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
	name, tags := splitNameAndTags(id)

	return []*Series{
		mr.counterI(name+".count", t, m.Count(), tags),
		mr.counterF(name+".min", t, millisI(m.Min()), tags),
		mr.counterF(name+".max", t, millisI(m.Max()), tags),
		mr.counterF(name+".mean", t, millisF(m.Mean()), tags),
		mr.counterF(name+".stddev", t, millisF(m.StdDev()), tags),
		mr.counterF(name+".median", t, millisF(ps[0]), tags),
		mr.counterF(name+".percentile.75", t, millisF(ps[1]), tags),
		mr.counterF(name+".percentile.95", t, millisF(ps[2]), tags),
		mr.counterF(name+".percentile.99", t, millisF(ps[3]), tags),
		mr.counterF(name+".percentile.999", t, millisF(ps[4]), tags),
		mr.counterF(name+".rate.1min", t, m.Rate1(), tags),
		mr.counterF(name+".rate.5min", t, m.Rate5(), tags),
		mr.counterF(name+".rate.15min", t, m.Rate15(), tags),
		mr.counterF(name+".rate.mean", t, m.RateMean(), tags),
	}
}

// `time.Duration` objects are always stored in nanoseconds. Here, we'll cast to
// floating point milliseconds to ease of understanding what's going on from the
// UI.
func millisI(nanos int64) float64 {
	return millisF(float64(nanos))
}
func millisF(nanos float64) float64 {
	return nanos / float64(time.Millisecond)
}

// func floatMs(nanos float64) int64 {
// 	return int64(nanos) / int64(time.Millisecond)
// }

func (mr *MetricsReporter) counterF(
	metric string, t int64, v float64, tags []string) *Series {
	return mr.seriesF(metric, "counter", t, v, tags)
}

func (mr *MetricsReporter) counterI(
	metric string, t int64, v int64, tags []string) *Series {
	return mr.seriesI(metric, "counter", t, v, tags)
}

func (mr *MetricsReporter) gaugeI(
	metric string, t int64, v int64, tags []string) *Series {
	return mr.seriesI(metric, "gauge", t, v, tags)
}

func (mr *MetricsReporter) seriesF(
	metric, typ string, t int64, v float64, tags []string) *Series {
	return &Series{
		Metric: metric,
		Points: [][2]interface{}{[2]interface{}{t, v}},
		Type:   typ,
		Host:   mr.client.Host,
		Tags:   tags,
	}
}

func (mr *MetricsReporter) seriesI(
	metric, typ string, t int64, v int64, tags []string) *Series {
	return &Series{
		Metric: metric,
		Points: [][2]interface{}{[2]interface{}{t, v}},
		Type:   typ,
		Host:   mr.client.Host,
		Tags:   tags,
	}
}

func splitNameAndTags(metric string) (string, []string) {
	if res := tagPattern.FindStringSubmatch(metric); len(res) == 3 {
		return res[1], strings.Split(res[2], ",")
	}
	return metric, make([]string, 0)
}
