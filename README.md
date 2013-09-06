# Go Datadog
Simple [Go](http://golang.org/) interface to the [Datadog
API](http://docs.datadoghq.com/api/).


## Metrics
The client can report metrics from
[go-metrics](https://github.com/rcrowley/go-metrics). Using either its
DefaultRegistry, or a custom one, metrics can be periodically sent with code
along the lines of the following:

```go
import(
  "github.com/vistarmedia/datadog"
  "os"
  "time"
)

host _ := os.Hostname()
dog := datadog.New(host, "dog-api-key")
go dog.DefaultReporter().Start(60 * time.Second)
```

And to use a custom registry, it would simply read:

```go
host _ := os.Hostname()
dog := datadog.New(host, "dog-api-key")
reporter := getMyCustomRegistry()
go dog.Reporter(registry).Start(60 * time.Second)
```
