package fasthttpprometheus

import (
	"fmt"
	"testing"

	"github.com/buaazp/fasthttprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var routesMap = map[string][]string{
	"GET": {
		"/ping",
		"/reload",
		"/user/:id",
		"/article/some-action/:id",
		"/config/:type/reload",
		"/user/:id/method",
		"/user/:id/not-method",
	},
	"POST": {
		"/user/:id",
	},
	"DELETE": {
		"/user/:id",
	},
}

var h = NewHandler(fasthttprouter.New(), "testing_service", zap.NewExample())
var registered bool

func Benchmark(b *testing.B) {
	if !registered {
		handleFunc := func(ctx *fasthttp.RequestCtx) {
			ctx.SuccessString("text/plain; charset=utf-8", "OK")
			return
		}
		for method, urls := range routesMap {
			for _, url := range urls {
				if method == "GET" {
					h.GET(url, handleFunc)
				}
				if method == "POST" {
					h.POST(url, handleFunc)
				}
				if method == "DELETE" {
					h.DELETE(url, handleFunc)
				}
			}
		}

		registered = true
	}

	for i := 0; i < b.N; i++ {
		for method, urls := range routesMap {
			for _, url := range urls {
				header := fasthttp.RequestHeader{}
				header.SetMethod(method)

				uri := &fasthttp.URI{}
				uri.SetPath(url)

				req := fasthttp.Request{}
				req.SetURI(uri)
				req.Header = header

				ctx := &fasthttp.RequestCtx{
					Request: req,
				}
				h.Handler(ctx)
			}
		}
	}
}

func TestProcessMetricName(t *testing.T) {
	var metricName string
	processMetricName("article", &metricName)
	assert.Equal(t, "article", metricName)

	processMetricName("some-action", &metricName)
	assert.Equal(t, "article_some_action", metricName)

	processMetricName(":id", &metricName)
	assert.Equal(t, "article_some_action", metricName)
}

type handlerSuite struct {
	suite.Suite

	obs     *observer.ObservedLogs
	handler *handler
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, &handlerSuite{})
}

func (s *handlerSuite) SetupTest() {
	var core zapcore.Core
	core, s.obs = observer.New(zap.InfoLevel)
	s.handler = NewHandler(fasthttprouter.New(), "test_service", zap.New(core))
}

func (s *handlerSuite) TestSetMetrics() {
	leaf := node{path: "method-one"}
	s.handler.setMetrics(&leaf, "GET", "user_method_one")

	s.Equal(
		"Desc{fqName: \"test_service_user_method_one_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_user_method_one_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)
}

func (s *handlerSuite) TestPutMethod() {
	s.handler.putMethod("/user/:id", "GET")
	s.handler.putMethod("/user/:id", "POST")
	s.handler.putMethod("/user/:id", "DELETE")
	s.handler.putMethod("/user/:id/some-method-one", "GET")
	s.handler.putMethod("/user/:id/some-method-two", "GET")
	s.handler.putMethod("/ping", "GET")
	s.handler.putMethod("/article/some-action/:id", "GET")

	leaf := s.handler.trie["GET"].getLeaf("/user/:id")
	s.Equal(":id", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_user_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_user_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)

	leaf = s.handler.trie["POST"].getLeaf("/user/:id")
	s.Equal(":id", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_user_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"POST\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_user_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"POST\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)

	leaf = s.handler.trie["DELETE"].getLeaf("/user/:id")
	s.Equal(":id", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_user_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"DELETE\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_user_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"DELETE\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)

	leaf = s.handler.trie["GET"].getLeaf("/user/:id/some-method-one")
	s.Equal("some-method-one", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_user_some_method_one_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_user_some_method_one_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)

	leaf = s.handler.trie["GET"].getLeaf("/ping")
	s.Equal("ping", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_ping_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_ping_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)

	leaf = s.handler.trie["GET"].getLeaf("/user/:id/some-method-two")
	s.Equal("some-method-two", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_user_some_method_two_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_user_some_method_two_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)

	leaf = s.handler.trie["GET"].getLeaf("/article/some-action/:id")
	s.Equal(":id", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_article_some_action_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_article_some_action_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)
}

func (s *handlerSuite) TestGET() {
	s.handler.putMethod("/something/:id", "GET")
	leaf := s.handler.trie["GET"].getLeaf("/something/:id")
	s.Equal(":id", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"GET\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)
}

func (s *handlerSuite) TestHEAD() {
	s.handler.putMethod("/something/:id", "HEAD")
	leaf := s.handler.trie["HEAD"].getLeaf("/something/:id")
	s.Equal(":id", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"HEAD\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"HEAD\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)
}

func (s *handlerSuite) TestOPTIONS() {
	s.handler.putMethod("/something/:id", "OPTIONS")
	leaf := s.handler.trie["OPTIONS"].getLeaf("/something/:id")
	s.Equal(":id", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"OPTIONS\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"OPTIONS\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)
}

func (s *handlerSuite) TestPOST() {
	s.handler.putMethod("/something/:id", "POST")
	leaf := s.handler.trie["POST"].getLeaf("/something/:id")
	s.Equal(":id", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"POST\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"POST\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)
}

func (s *handlerSuite) TestPUT() {
	s.handler.putMethod("/something/:id", "PUT")
	leaf := s.handler.trie["PUT"].getLeaf("/something/:id")
	s.Equal(":id", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"PUT\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"PUT\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)
}

func (s *handlerSuite) TestPATCH() {
	s.handler.putMethod("/something/:id", "PATCH")
	leaf := s.handler.trie["PATCH"].getLeaf("/something/:id")
	s.Equal(":id", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"PATCH\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"PATCH\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)
}

func (s *handlerSuite) TestDELETE() {
	s.handler.putMethod("/something/:id", "DELETE")
	leaf := s.handler.trie["DELETE"].getLeaf("/something/:id")
	s.Equal(":id", leaf.path)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_total\", help: \"\", "+
			"constLabels: {http_method=\"DELETE\"}, variableLabels: []}",
		leaf.metrics[metricTypeTotal].Desc().String(),
	)
	s.Equal(
		"Desc{fqName: \"test_service_something_requests_failure\", help: \"\", "+
			"constLabels: {http_method=\"DELETE\"}, variableLabels: []}",
		leaf.metrics[metricTypeFailure].Desc().String(),
	)
}

func (s *handlerSuite) TestIncNotFound() {
	metrics := map[string]prometheus.Counter{}
	err := s.handler.inc(metrics, metricTypeTotal)

	s.Equal(metricNotFoundErr, err)
}

func (s *handlerSuite) TestIncOk() {
	metrics := map[string]prometheus.Counter{
		metricTypeTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name:      fmt.Sprintf("%s_%s_%s", "metricName", requests, metricTypeTotal),
			Namespace: h.service,
			ConstLabels: prometheus.Labels{
				"http_method": "GET",
			},
		}),
	}
	err := s.handler.inc(metrics, metricTypeTotal)

	s.Nil(err)
}

func (s *handlerSuite) TestLibHandlerFindTreeErr() {
	header := fasthttp.RequestHeader{}
	header.SetMethod("GET")

	req := fasthttp.Request{}
	req.Header = header
	s.handler.libHandler(&fasthttp.RequestCtx{
		Request: req,
	})

	s.Equal(
		1,
		s.obs.FilterMessage("can't find tree").
			FilterField(zap.ByteString("http_method", []byte("GET"))).
			Len(),
	)
}

func (s *handlerSuite) TestLibHandlerFindLeafErr() {
	s.handler.putMethod("/some-path-for-leaf-err", "GET")

	header := fasthttp.RequestHeader{}
	header.SetMethod("GET")

	uri := &fasthttp.URI{}
	uri.SetPath("/find-leaf")

	req := fasthttp.Request{}
	req.SetURI(uri)
	req.Header = header
	s.handler.libHandler(&fasthttp.RequestCtx{
		Request: req,
	})

	s.Equal(
		1,
		s.obs.FilterMessage("can't find leaf for path").
			FilterField(zap.ByteString("path", []byte("/find-leaf"))).
			FilterField(zap.ByteString("http_method", []byte("GET"))).
			Len(),
	)
}

func (s *handlerSuite) TestLibHandlerIncTotalMetricErr() {
	s.handler.putMethod("/some-path-for-total-metric-err", "GET")
	leaf := s.handler.trie["GET"].getLeaf("/some-path-for-total-metric-err")
	leaf.metrics = nil

	header := fasthttp.RequestHeader{}
	header.SetMethod("GET")

	uri := &fasthttp.URI{}
	uri.SetPath("/some-path-for-total-metric-err")

	req := fasthttp.Request{}
	req.SetURI(uri)
	req.Header = header
	s.handler.libHandler(&fasthttp.RequestCtx{
		Request: req,
	})

	s.Equal(
		1,
		s.obs.FilterMessage("can't find metric").
			FilterField(zap.ByteString("path", []byte("/some-path-for-total-metric-err"))).
			FilterField(zap.ByteString("http_method", []byte("GET"))).
			FilterField(zap.String("metric_type", metricTypeTotal)).
			FilterField(zap.Any("metrics", leaf.metrics)).
			Len(),
	)
}

func (s *handlerSuite) TestLibHandlerIncFailureMetricErr() {
	s.handler.putMethod("/some-path-for-failure-metric-err", "GET")
	leaf := s.handler.trie["GET"].getLeaf("/some-path-for-failure-metric-err")
	delete(leaf.metrics, metricTypeFailure)

	header := fasthttp.RequestHeader{}
	header.SetMethod("GET")

	uri := &fasthttp.URI{}
	uri.SetPath("/some-path-for-failure-metric-err")

	req := fasthttp.Request{}
	req.SetURI(uri)
	req.Header = header

	resp := fasthttp.Response{}
	resp.SetStatusCode(fasthttp.StatusInternalServerError)

	s.handler.libHandler(&fasthttp.RequestCtx{
		Request:  req,
		Response: resp,
	})

	s.Equal(
		1,
		s.obs.FilterMessage("can't find metric").
			FilterField(zap.ByteString("path", []byte("/some-path-for-failure-metric-err"))).
			FilterField(zap.ByteString("http_method", []byte("GET"))).
			FilterField(zap.String("metric_type", metricTypeFailure)).
			FilterField(zap.Any("metrics", leaf.metrics)).
			Len(),
	)
}

func (s *handlerSuite) TestLibHandlerOk() {
	s.handler.putMethod("/some-path-for-metric-ok", "GET")

	header := fasthttp.RequestHeader{}
	header.SetMethod("GET")

	uri := &fasthttp.URI{}
	uri.SetPath("/some-path-for-metric-ok")

	req := fasthttp.Request{}
	req.SetURI(uri)
	req.Header = header

	s.handler.libHandler(&fasthttp.RequestCtx{
		Request: req,
	})

	s.Equal(0, s.obs.FilterMessage("can't find metric").Len())
}
