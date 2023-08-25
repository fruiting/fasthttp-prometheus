# fasthttp-prometheus
Prometheus metrics exporter for fasthttp. On every method creates two metrics: total and failure.
For example you want to register path `/user/:id/some-method` in your fasthttp server.
Library will create metrics based on [OpenMetrics](https://github.com/OpenObservability/OpenMetrics/tree/main):
1. `{prefix}_user_some_method_requests_total`
2. `{prefix}_user_some_method_requests_failure_total`

## Installation
```
go get github.com/fruiting/fasthttp-prometheus
```

## Usage
```
import (
    "github.com/buaazp/fasthttprouter"
    fasthttpprometheus "github.com/fruiting/fasthttp-prometheus"
    "github.com/valyala/fasthttp"
    "go.uber.org/zap"
)

func main() {
    wrappedRouter := fasthttpprometheus.NewHandler(fasthttprouter.New(), "test_service", zap.NewExample())
    wrappedRouter.GET("/ping", func(ctx *fasthttp.RequestCtx) {
        ctx.SuccessString("text/plain; charset=utf-8", "PONG")
    })

    fasthttp.ListenAndServe(":8080", wrappedRouter.Handler)
}
```

## Benchmarking
Benchmark shows about 10% speed reduction of fasthttp.
On MacBook M1 Pro on the same list of registered routes fasthttp shows 8900-9200 ns/op
and 9600-10000 ns/op if fasthttp is wrapped by this library.

## Contribute
1. Run unit tests `go test ./...`
2. Run benchmarks `go test -bench=.`
3. Push, make pull request

## Licence
[MIT](https://github.com/fruiting/fasthttp-prometheus/blob/master/LICENSE)

### MAINTAINER
Roman Spirin
