package proxy

import (
	"context"
	"fmt"
	"google.golang.org/grpc/peer"
	"local-webhook-tester/transport"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type ReverseProxyServer struct {
	serverBaseUrl string
	httpRequests  map[string]chan transportRequestWithContext
	httpResponses map[string]chan *transport.HttpResponse
	roundTrippers map[string]http.RoundTripper
	logger        *log.Logger
	transport.UnimplementedHttpReverseProxyServer
}

type transportRequestWithContext struct {
	httpRequest *transport.HttpRequest
	ctx         context.Context
}

type HttpForwardProxy interface {
	ProxyRequest(ctx context.Context, request *http.Request) <-chan *transport.HttpResponse
}

type UrlInUseError struct {
	PathPrefix string
}

func (u UrlInUseError) Error() string {
	return fmt.Sprintf("Path prefix %s already in use", u.PathPrefix)
}

func NewReverseProxy(config *ServerConfig) ReverseProxyServer {
	requests := make(map[string]chan transportRequestWithContext)
	responses := make(map[string]chan *transport.HttpResponse)
	logger := log.New(os.Stdout, "[grpc] ", log.LstdFlags)
	return ReverseProxyServer{
		serverBaseUrl:                       config.BaseUrl,
		httpRequests:                        requests,
		httpResponses:                       responses,
		UnimplementedHttpReverseProxyServer: transport.UnimplementedHttpReverseProxyServer{},
		logger:                              logger,
		roundTrippers:                       map[string]http.RoundTripper{},
	}
}

func (r ReverseProxyServer) ReverseProxy(server transport.HttpReverseProxy_ReverseProxyServer) error {
	proxy, err := setupProxy(server, r.serverBaseUrl, r.logger)
	if err != nil {
		return err
	}
	proxy.Run()

	r.roundTrippers[proxy.Prefix()] = GrpcProxyRoundTripper{proxy: proxy}

	<-proxy.Context().Done()
	delete(r.roundTrippers, proxy.Prefix())
	return nil
}

func (r ReverseProxyServer) GetHttpTransport() PrefixRoundTripper {
	return PrefixRoundTripper{roundTrippers: r.roundTrippers}
}

func setupProxy(server transport.HttpReverseProxy_ReverseProxyServer, serverBaseUrl string, logger *log.Logger) (GrpcHttpProxy, error) {
	baseUrl, err := url.Parse(serverBaseUrl)
	if err != nil {
		return nil, err
	}
	prefix := generateRandomUrlPrefix()
	p, _ := peer.FromContext(server.Context())
	logger.Printf("Starting new proxy on %s for %v", prefix, p.Addr)
	err = sendProxyUrl(server, baseUrl, prefix, err)

	return NewGrpcHttpProxy(prefix, server, logger)
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

func generateRandomUrlPrefix() string {
	return strconv.Itoa(int(rand.Int31()))
}

func (r ReverseProxyServer) mustEmbedUnimplementedHttpReverseProxyServer() {
	panic("implement me")
}
