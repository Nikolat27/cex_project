package HttpServer

import (
	"cex_project/internal/handlers"
	"fmt"
	"net/http"
)

type HttpServer struct {
	server *http.Server
	port string
}

func NewHttpServer(port string, handlers *handlers.Handlers) *HttpServer {
	router := newCoreRouter(handlers)

	return &HttpServer{
		server: &http.Server{
			Addr: fmt.Sprintf(":%s", port),
			Handler: router.Router,
		},
		port: port,
	}
}

func (h *HttpServer) Start() error {
	return h.server.ListenAndServe()
}

func (h *HttpServer) Port() string {
	return h.port
}

func (h *HttpServer) Close() error {
	return h.server.Close()
}
