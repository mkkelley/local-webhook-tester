package proxy

import (
	"fmt"
	"local-webhook-tester/transport"
	"log"
	"net/url"
	"os"
)

type ReverseProxy struct {
	serverBaseUrl string
	httpRequests  map[string]chan *transport.HttpRequest
	httpResponses map[string]chan *transport.HttpResponse
	logger        *log.Logger
	transport.UnimplementedHttpReverseProxyServer
}

type UrlInUseError struct {
	PathPrefix string
}

func (u UrlInUseError) Error() string {
	return fmt.Sprintf("Path prefix %s already in use", u.PathPrefix)
}

func NewReverseProxy(config *ServerConfig) ReverseProxy {
	requests := make(map[string]chan *transport.HttpRequest)
	responses := make(map[string]chan *transport.HttpResponse)
	logger := log.New(os.Stdout, "[grpc] ", log.LstdFlags)
	return ReverseProxy{
		serverBaseUrl:                       config.BaseUrl,
		httpRequests:                        requests,
		httpResponses:                       responses,
		UnimplementedHttpReverseProxyServer: transport.UnimplementedHttpReverseProxyServer{},
		logger:                              logger,
	}
}

func (r ReverseProxy) ReverseProxy(server transport.HttpReverseProxy_ReverseProxyServer) error {
	baseUrl, err := url.Parse(r.serverBaseUrl)
	if err != nil {
		return err
	}
	prefix := generateRandomUrlPrefix()
	r.logger.Printf("Starting new proxy on %s", prefix)
	err = sendProxyUrl(server, baseUrl, prefix, err)
	if err != nil {
		return err
	}

	requestCh := r.httpRequests[prefix]
	if requestCh != nil {
		return UrlInUseError{PathPrefix: prefix}
	}

	requestCh = make(chan *transport.HttpRequest)
	r.httpRequests[prefix] = requestCh

	responseCh := r.httpResponses[prefix]
	if responseCh != nil {
		return UrlInUseError{PathPrefix: prefix}
	}

	responseCh = make(chan *transport.HttpResponse)
	r.httpResponses[prefix] = responseCh

	for {
		request := <-requestCh
		r.logger.Printf("Got request %s", request.Path)
		requestResponse := &transport.ReverseProxyResponse_HttpRequest{HttpRequest: request}
		err = server.Send(&transport.ReverseProxyResponse{Response: requestResponse})
		if err != nil {
			return err
		}

		httpResponse := &transport.HttpResponse{}
		err = server.RecvMsg(httpResponse)
		r.logger.Printf("Got response %d", httpResponse.ResponseCode)
		if err != nil {
			return err
		}

		responseCh <- httpResponse
	}
}

func sendProxyUrl(server transport.HttpReverseProxy_ReverseProxyServer, baseUrl *url.URL, prefix string, err error) error {
	baseUrl.Path = prefix
	urlString := baseUrl.String()

	startResponse := &transport.ProxyStartResponse{BaseUrl: urlString}
	err = server.SendMsg(
		&transport.ReverseProxyResponse{
			Response: &transport.ReverseProxyResponse_ProxyStartResponse{
				ProxyStartResponse: startResponse,
			},
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (r ReverseProxy) mustEmbedUnimplementedHttpReverseProxyServer() {
	panic("implement me")
}
