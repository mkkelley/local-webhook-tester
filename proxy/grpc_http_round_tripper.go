package proxy

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type ServiceUnavailableError struct {
	prefix string
}

func (s ServiceUnavailableError) Error() string {
	return fmt.Sprintf("Service not found with prefix: %v", s.prefix)
}

type PrefixRoundTripper struct {
	roundTrippers map[string]http.RoundTripper
}

func (p PrefixRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	prefix := chi.URLParam(request, "proxyBase")
	path := "/" + chi.URLParam(request, "*")
	request.URL.Path = path

	roundTripper := p.roundTrippers[prefix]
	if roundTripper == nil {
		return nil, ServiceUnavailableError{prefix: prefix}
	}
	return p.roundTrippers[prefix].RoundTrip(request)
}

type GrpcProxyRoundTripper struct {
	proxy GrpcHttpProxy
}

func (h GrpcProxyRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	serializedRequest, err := serializeHttpRequest(request)
	if err != nil {
		return nil, err
	}

	requestId, err := h.proxy.SubmitRequest(request.Context(), serializedRequest)
	if err != nil {
		return nil, err
	}
	response, err := h.proxy.AwaitResponse(request.Context(), requestId)
	if err != nil {
		return nil, err
	}

	httpResponse := deserializeGrpcHttpResponse(request, response)
	return httpResponse, nil
}
