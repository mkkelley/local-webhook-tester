package proxy

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"net/url"
	"time"
)

func RunHttpServer(config *ServerConfig, proxyServer *ReverseProxyServer) error {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Minute))

	r.Handle("/{proxyBase}/*", handleRequest(proxyServer))
	return http.ListenAndServe(fmt.Sprintf(":%s", config.HttpPort), r)
}

func handleRequest(proxy *ReverseProxyServer) http.Handler {
	roundTripper := proxy.GetHttpTransport()
	httpClient := http.Client{
		Transport: roundTripper,
		Timeout:   5 * time.Minute,
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var err error
		request.URL, err = url.Parse(request.RequestURI)
		request.RequestURI = ""
		if err != nil {
			http.Error(writer, fmt.Sprintf("Error encoding request URL: %v", err), http.StatusInternalServerError)
			return
		}

		response, err := httpClient.Do(request)
		if err != nil {
			http.Error(writer, fmt.Sprintf("Error executing request: %v", err), http.StatusBadGateway)
			return
		}

		err = deserializeHttpResponseToResponseWriter(writer, response)
		if err != nil {
			http.Error(writer, fmt.Sprintf("error writing response: %v", err), http.StatusInternalServerError)
		}
	})
}
