package proxy

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"io"
	"local-webhook-tester/transport"
	"local-webhook-tester/util"
	"net/http"
	"strings"
	"time"
)

func RunHttpServer(config *ServerConfig, proxyServer *ReverseProxy) error {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Minute))

	r.Handle("/{proxyBase}/*", handleRequest(proxyServer))
	return http.ListenAndServe(fmt.Sprintf(":%s", config.HttpPort), r)
}

func convertRequest(request *http.Request) (*transport.HttpRequest, error) {
	path := "/" + chi.URLParam(request, "*")
	bodyString, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	headers := util.SerializeHeader(request.Header)

	var fullPath string
	if request.URL.RawQuery == "" {
		fullPath = path
	} else {
		fullPath = fmt.Sprintf("%s?%s", path, request.URL.RawQuery)
	}
	return &transport.HttpRequest{
		Method:  request.Method,
		Path:    fullPath,
		Body:    string(bodyString),
		Headers: headers,
	}, nil
}

func writeResponse(writer http.ResponseWriter, response *transport.HttpResponse) error {
	for _, header := range response.Headers {
		split := strings.Split(header, ":")
		key := split[0]
		val := split[1][1:]

		writer.Header().Add(key, val)
	}

	writer.WriteHeader(int(response.ResponseCode))
	_, err := writer.Write([]byte(response.Body))
	return err
}

func handleRequest(proxy *ReverseProxy) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		proxyBase := chi.URLParam(request, "proxyBase")
		if proxyBase == "" {
			http.Error(writer, "unspecified service", 503)
			return
		}

		requestCh := proxy.httpRequests[proxyBase]
		responseCh := proxy.httpResponses[proxyBase]
		if requestCh == nil || responseCh == nil {
			http.Error(writer, "specified service does not exist", http.StatusServiceUnavailable)
			return
		}

		transportRequest, err := convertRequest(request)
		if err != nil {
			http.Error(writer, "error converting request", http.StatusInternalServerError)
			return
		}
		requestCh <- transportRequest
		response := <-responseCh
		err = writeResponse(writer, response)
		if err != nil {
			http.Error(writer, "error writing response", http.StatusInternalServerError)
		}
	})
}
