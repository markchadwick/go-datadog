package datadog

import (
	"io/ioutil"
	. "launchpad.net/gocheck"
	"strings"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})
var client *Client

func (s *ClientSuite) SetUpTest(c *C) {
	client = &Client{}
}

func (s *ClientSuite) TestSeriesEndpoint(c *C) {
	client.ApiKey = "secret"
	c.Check(client.SeriesUrl(), Equals,
		"https://app.datadoghq.com/api/v1/series?api_key=secret")
}

func (s *ClientSuite) TestSingleSeriesReader(c *C) {
	series := &Series{
		Metric: "foo.bar.baz",
		Points: [][2]interface{}{[2]interface{}{1346340794, 66.6}},
		Type:   "gauge",
		Host:   "hostname",
		Tags:   []string{"one", "two", "three"},
	}

	reader, err := client.seriesReader([]*Series{series})
	c.Check(err, IsNil)

	b, err := ioutil.ReadAll(reader)
	c.Check(err, IsNil)

	body := string(b)
	c.Check(strings.Index(body, `"metric":"foo.bar.baz"`), Not(Equals), -1)
	c.Check(strings.Index(body, `"points":[[1346340794,66.6]]`), Not(Equals), -1)
	c.Check(strings.Index(body, `"type":"gauge"`), Not(Equals), -1)
	c.Check(strings.Index(body, `"host":"hostname"`), Not(Equals), -1)
	c.Check(strings.Index(body, `"tags":["one","two","three"]`), Not(Equals), -1)
}
