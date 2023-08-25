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
	// every metric name part
	requests string = "requests"
	// metric type
	metricTypeTotal string = "total"
	// metric type
	metricTypeFailure string = "failure_total"
)

const (
	// byte for symbol "-"
	dashByte uint8 = 45
	// byte for symbol "/"
	slashByte uint8 = 47
	// byte for symbol ":"
	colonByte uint8 = 58
	// byte for symbol "_"
	underlineByte uint8 = 95
)

var (
	metricNotFoundErr = errors.New("metric not found")
)

func processMetricName(path string, metricName *string) {
	if path[1] == colonByte {
		if path[len(path)-1] == slashByte {
			*metricName = *metricName + "_" + path[2:len(path)-1] + "_var"
		} else {
			*metricName = *metricName + "_" + path[2:] + "_var"
		}
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

	if bytes[0] == slashByte {
		bytes = bytes[1:]
	}
	if bytes[len(bytes)-1] == slashByte {
		bytes = bytes[:len(bytes)-1]
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
			h.logger.Error(
				"libfasthttp-prometheus recovered from panic",
				zap.String("panic_msg", fmt.Sprintf("%v", r)),
			)
		}
	}()

	root, ok := h.trie[httpMethod]
	if !ok {
		root = new(node)
		h.trie[httpMethod] = root
	}

	var metricName string
	leaf := root.addPath(path, &metricName)
	h.setMetrics(
		leaf,
		h.createMetric(metricName, httpMethod, metricTypeTotal),
		h.createMetric(metricName, httpMethod, metricTypeFailure),
	)
}

func (h *handler) createMetric(metricName, httpMethod, metricType string) prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{
		Name:      fmt.Sprintf("%s_%s_%s", metricName, requests, metricType),
		Namespace: h.service,
		ConstLabels: prometheus.Labels{
			"http_method": httpMethod,
		},
	})
}

func (h *handler) setMetrics(leaf *node, metricTotal, metricFailure prometheus.Counter) {
	leaf.metrics = map[string]prometheus.Counter{
		metricTypeTotal:   metricTotal,
		metricTypeFailure: metricFailure,
	}

	err := prometheus.Register(metricTotal)
	if err != nil {
		h.logger.Warn("can't register total metric", zap.Error(err))

		return
	}

	err = prometheus.Register(metricFailure)
	if err != nil {
		h.logger.Warn("can't register failure metric", zap.Error(err))
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
		return
	}

	err := h.inc(leaf.metrics, metricTypeTotal)
	if err != nil {
		h.logger.Warn(
			"can't find metric",
			zap.ByteString("path", ctx.URI().Path()),
			zap.ByteString("http_method", ctx.Method()),
			zap.String("metric_type", metricTypeTotal),
		)

		return
	}

	// if status_code >= 400 it will be marked as error and increment fail metric
	if ctx.Response.StatusCode() >= fasthttp.StatusBadRequest {
		err = h.inc(leaf.metrics, metricTypeFailure)
		if err != nil {
			h.logger.Warn(
				"can't find metric",
				zap.ByteString("path", ctx.URI().Path()),
				zap.ByteString("http_method", ctx.Method()),
				zap.String("metric_type", metricTypeFailure),
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
