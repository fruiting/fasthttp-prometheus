package fasthttpprometheus

import (
	"errors"
	"fmt"

	"github.com/buaazp/fasthttprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

const (
	requests string = "requests"

	metricTypeTotal   string = "total"
	metricTypeFailure string = "failure"

	// byte for "-" symbol
	dashByte uint8 = 45
	// byte for "/" symbol
	slashByte uint8 = 47
	// byte for ":" symbol
	colonByte uint8 = 58
	// byte for "_" symbol
	underlineByte uint8 = 95
)

var (
	metricNotFoundErr = errors.New("metric not found")
)

func processMetricName(path string, metricName *string) {
	if path[0] == colonByte {
		return
	}

	bytes := make([]byte, len(path), len(path))
	for i := 0; i < len(path); i++ {
		if path[i] == dashByte {
			bytes[i] = underlineByte
			continue
		}

		bytes[i] = path[i]
	}

	if len(*metricName) == 0 {
		*metricName += string(bytes)
	} else {
		*metricName = *metricName + "_" + string(bytes)
	}
}

type handler struct {
	router  *fasthttprouter.Router
	service string
	trie    map[string]*node
	logger  *zap.Logger
}

func NewHandler(router *fasthttprouter.Router, service string, logger *zap.Logger) *handler {
	return &handler{
		router:  router,
		service: service,
		trie:    make(map[string]*node, 0),
		logger:  logger,
	}
}

func (h *handler) Handler(ctx *fasthttp.RequestCtx) {
	h.router.Handler(ctx)
	h.libHandler(ctx)
}

func (h *handler) GET(path string, handle fasthttp.RequestHandler) {
	h.putMethod(path, "GET")
	h.router.GET(path, handle)
}

func (h *handler) HEAD(path string, handle fasthttp.RequestHandler) {
	h.putMethod(path, "HEAD")
	h.router.HEAD(path, handle)
}

func (h *handler) OPTIONS(path string, handle fasthttp.RequestHandler) {
	h.putMethod(path, "OPTIONS")
	h.router.OPTIONS(path, handle)
}

func (h *handler) POST(path string, handle fasthttp.RequestHandler) {
	h.putMethod(path, "POST")
	h.router.POST(path, handle)
}

func (h *handler) PUT(path string, handle fasthttp.RequestHandler) {
	h.putMethod(path, "PUT")
	h.router.PUT(path, handle)
}

func (h *handler) PATCH(path string, handle fasthttp.RequestHandler) {
	h.putMethod(path, "PATCH")
	h.router.PATCH(path, handle)
}

func (h *handler) DELETE(path string, handle fasthttp.RequestHandler) {
	h.putMethod(path, "DELETE")
	h.router.DELETE(path, handle)
}

func (h *handler) putMethod(path, httpMethod string) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("fasthttp-prometheus recovered from panic", zap.Any("error", r))
		}
	}()

	trees := h.trie
	root := trees[httpMethod]
	if root == nil {
		root = new(node)
		trees[httpMethod] = root
	}

	var metricName string
	leaf := root.addPath(path, &metricName)
	h.setMetrics(leaf, httpMethod, metricName)
}

func (h *handler) setMetrics(leaf *node, httpMethod, metricName string) {
	metrics := [2]prometheus.Counter{
		prometheus.NewCounter(prometheus.CounterOpts{
			Name:      fmt.Sprintf("%s_%s_%s", metricName, requests, metricTypeTotal),
			Namespace: h.service,
			ConstLabels: prometheus.Labels{
				"http_method": httpMethod,
			},
		}),
		prometheus.NewCounter(prometheus.CounterOpts{
			Name:      fmt.Sprintf("%s_%s_%s", metricName, requests, metricTypeFailure),
			Namespace: h.service,
			ConstLabels: prometheus.Labels{
				"http_method": httpMethod,
			},
		}),
	}

	prometheus.MustRegister(metrics[0], metrics[1])
	leaf.metrics = map[string]prometheus.Counter{
		metricTypeTotal:   metrics[0],
		metricTypeFailure: metrics[1],
	}
}

func (h *handler) libHandler(ctx *fasthttp.RequestCtx) {
	root, ok := h.trie[string(ctx.Method())]
	if !ok {
		h.logger.Error("can't find tree", zap.ByteString("http_method", ctx.Method()))

		return
	}

	leaf := root.getLeaf(string(ctx.URI().Path()))
	if leaf == nil {
		h.logger.Error(
			"can't find leaf for path",
			zap.ByteString("path", ctx.URI().Path()),
			zap.ByteString("http_method", ctx.Method()),
		)

		return
	}

	err := h.inc(leaf.metrics, metricTypeTotal)
	if err != nil {
		h.logger.Warn(
			"can't find metric",
			zap.ByteString("path", ctx.URI().Path()),
			zap.ByteString("http_method", ctx.Method()),
			zap.String("metric_type", metricTypeTotal),
			zap.Any("metrics", leaf.metrics),
		)

		return
	}

	if ctx.Response.StatusCode() >= fasthttp.StatusBadRequest {
		err = h.inc(leaf.metrics, metricTypeFailure)
		if err != nil {
			h.logger.Warn(
				"can't find metric",
				zap.ByteString("path", ctx.URI().Path()),
				zap.ByteString("http_method", ctx.Method()),
				zap.String("metric_type", metricTypeFailure),
				zap.Any("metrics", leaf.metrics),
			)

			return
		}
	}
}

func (h *handler) inc(metrics map[string]prometheus.Counter, metricType string) error {
	metric, ok := metrics[metricType]
	if !ok {
		return metricNotFoundErr
	}

	metric.Inc()

	return nil
}
