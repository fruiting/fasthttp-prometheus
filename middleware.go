package fasthttpprometheus

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

type Handler struct {
	Router  *fasthttprouter.Router
	methods []string
	mu      *sync.RWMutex
}

func NewHandler(router *fasthttprouter.Router) *Handler {
	return &Handler{
		Router: router,
		mu:     &sync.RWMutex{},
	}
}

func (h *Handler) Handler(ctx *fasthttp.RequestCtx) {
	h.Router.Handler(ctx)
	h.libHandler(ctx)
}

func (h *Handler) GET(path string, handle fasthttp.RequestHandler) {
	go h.putMethod(path)
	h.Router.GET(path, handle)
}

func (h *Handler) HEAD(path string, handle fasthttp.RequestHandler) {
	go h.putMethod(path)
	h.Router.HEAD(path, handle)
}

func (h *Handler) OPTIONS(path string, handle fasthttp.RequestHandler) {
	go h.putMethod(path)
	h.Router.OPTIONS(path, handle)
}

func (h *Handler) POST(path string, handle fasthttp.RequestHandler) {
	go h.putMethod(path)
	h.Router.POST(path, handle)
}

func (h *Handler) PUT(path string, handle fasthttp.RequestHandler) {
	go h.putMethod(path)
	h.Router.PUT(path, handle)
}

func (h *Handler) PATCH(path string, handle fasthttp.RequestHandler) {
	go h.putMethod(path)
	h.Router.PATCH(path, handle)
}

func (h *Handler) DELETE(path string, handle fasthttp.RequestHandler) {
	go h.putMethod(path)
	h.Router.DELETE(path, handle)
}

func (h *Handler) putMethod(path string) {
	ss := strings.Split(path, "/")
	reg, err := regexp.Compile(`^\{.*?\}+$`)
	if err != nil {
		//todo придумать как обработать
	}

	validParts := make([]string, 0, len(ss))
	for _, part := range ss {
		if !reg.Match([]byte(part)) {
			validParts = append(validParts, part)
		}
	}

	methodName := strings.Join(validParts, "/")

	//todo check
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, method := range h.methods {
		if method == methodName {
			return
		}
	}

	h.mu.Lock()
	h.methods = append(h.methods, methodName)
	h.mu.Unlock()
}

func (h *Handler) libHandler(ctx *fasthttp.RequestCtx) {
	fmt.Println("a")
	//todo тут будет хендлер от либы
}
