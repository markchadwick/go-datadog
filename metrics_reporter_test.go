package datadog

import (
	"github.com/rcrowley/go-metrics"
	. "launchpad.net/gocheck"
	"time"
)

type ReporterSuite struct{}

var (
	_        = Suite(&ReporterSuite{})
	registry metrics.Registry
	sample   metrics.Sample
	reporter *MetricsReporter
	t        time.Time
)

func (s *ReporterSuite) SetUpTest(c *C) {
	registry = metrics.NewRegistry()
	sample = metrics.NewUniformSample(1028)
	client = &Client{
		Host: "My Host",
	}
	reporter = &MetricsReporter{client, registry}
	t = time.Now()
}

func (s *ReporterSuite) TestSimpleReport(c *C) {
	counter := metrics.NewCounter()
	counter.Inc(666)
	meter := metrics.NewMeter()
	meter.Mark(222)
	meter.Mark(444)

	registry.Register("my.counter", counter)
	registry.Register("my.meter", meter)

	series := reporter.Series()
	c.Check(series, HasLen, 6)
	c.Check(series[0].Metric, Equals, "my.counter.count")
	c.Check(series[1].Metric, Equals, "my.meter.count")
	c.Check(series[2].Metric, Equals, "my.meter.rate.1min")
	c.Check(series[3].Metric, Equals, "my.meter.rate.5min")
	c.Check(series[4].Metric, Equals, "my.meter.rate.15min")
	c.Check(series[5].Metric, Equals, "my.meter.rate.mean")
}

func (_ *ReporterSuite) TestCounterSeries(c *C) {
	counter := metrics.NewCounter()
	counter.Inc(444)
	counter.Inc(222)
	series := reporter.series(t.Unix(), "my.counter", counter)
	c.Check(series, HasLen, 1)
	s := series[0]

	c.Check(s.Metric, Equals, "my.counter.count")
	c.Check(s.Type, Equals, "counter")
	c.Check(s.Points, HasLen, 1)
	c.Check(s.Points[0][0], Equals, t.Unix())
	c.Check(s.Points[0][1], Equals, int64(666))
}

func (_ *ReporterSuite) TestGaugeSeries(c *C) {
	gauge := metrics.NewGauge()
	gauge.Update(444)
	gauge.Update(222)

	series := reporter.series(t.Unix(), "my.gauge", gauge)
	c.Check(series, HasLen, 1)
	s := series[0]

	c.Check(s.Metric, Equals, "my.gauge.value")
	c.Check(s.Type, Equals, "gauge")
	c.Check(s.Points, HasLen, 1)
	c.Check(s.Points[0][0], Equals, t.Unix())
	c.Check(s.Points[0][1], Equals, int64(222))
}

func (_ *ReporterSuite) TestHealthcheckSeries(c *C) {
	c.Skip("Healthchecks presently not impelented")
}

func (_ *ReporterSuite) TestHistogramSeries(c *C) {
	hist := metrics.NewHistogram(sample)
	hist.Update(1)
	hist.Update(2)
	hist.Update(4)
	hist.Update(8)
	hist.Update(16)

	series := reporter.series(t.Unix(), "my.hist", hist)
	c.Check(series, HasLen, 10)

	c.Check(series[0].Metric, Equals, "my.hist.count")
	c.Check(series[0].Points[0][1], Equals, int64(5))

	c.Check(series[1].Metric, Equals, "my.hist.min")
	c.Check(series[1].Points[0][1], Equals, int64(1))

	c.Check(series[2].Metric, Equals, "my.hist.max")
	c.Check(series[2].Points[0][1], Equals, int64(16))

	c.Check(series[3].Metric, Equals, "my.hist.mean")
	c.Check(series[3].Points[0][1], Equals, 6.2)

	c.Check(series[4].Metric, Equals, "my.hist.stddev")

	c.Check(series[5].Metric, Equals, "my.hist.median")
	c.Check(series[5].Points[0][1], Equals, float64(4))

	c.Check(series[6].Metric, Equals, "my.hist.percentile.75")
	c.Check(series[6].Points[0][1], Equals, float64(12))

	c.Check(series[7].Metric, Equals, "my.hist.percentile.95")
	c.Check(series[7].Points[0][1], Equals, float64(16))

	c.Check(series[8].Metric, Equals, "my.hist.percentile.99")
	c.Check(series[8].Points[0][1], Equals, float64(16))

	c.Check(series[9].Metric, Equals, "my.hist.percentile.999")
	c.Check(series[9].Points[0][1], Equals, float64(16))
}

func (_ *ReporterSuite) TestMeterSeries(c *C) {
	meter := metrics.NewMeter()
	meter.Mark(222)
	meter.Mark(444)

	series := reporter.series(t.Unix(), "my.meter", meter)
	c.Check(series, HasLen, 5)

	c.Check(series[0].Metric, Equals, "my.meter.count")
	c.Check(series[0].Points[0][1], Equals, int64(666))
}

func (_ *ReporterSuite) TestTimerSeries(c *C) {
	timer := metrics.NewTimer()
	timer.Time(func() {
		time.Sleep(23 * time.Millisecond)
	})

	series := reporter.series(t.Unix(), "my.timer", timer)
	c.Check(series, HasLen, 14)

	c.Check(series[0].Metric, Equals, "my.timer.count")
	c.Check(series[0].Points[0][1], Equals, int64(1))

	c.Check(series[1].Metric, Equals, "my.timer.min")
	min, ok := series[1].Points[0][1].(float64)
	c.Check(ok, Equals, true)
	// Make sure timestamps have been converted to milliseconds.
	c.Check(min >= 23, Equals, true)
	c.Check(min < 30, Equals, true)

	c.Check(series[2].Metric, Equals, "my.timer.max")
}
