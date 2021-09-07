package proxy

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"io"
	"local-webhook-tester/transport"
	"log"
	"math/rand"
	"net"
)

type GrpcHttpProxy interface {
	SubmitRequest(ctx context.Context, request *transport.HttpRequest) (int64, error)
	AwaitResponse(ctx context.Context, requestId int64) (*transport.HttpResponse, error)
	Prefix() string
	Context() context.Context
	Run()
}

type grpcHttpProxyImpl struct {
	prefix   string
	peerAddr net.Addr

	serverStream transport.HttpReverseProxy_ReverseProxyServer
	ctx          context.Context

	requestCh        chan *transport.HttpRequest
	responseChannels map[int64]chan responseWithError
	logger           *log.Logger
}

func (g grpcHttpProxyImpl) Prefix() string {
	return g.prefix
}

func (g grpcHttpProxyImpl) Context() context.Context {
	return g.ctx
}

type responseWithError struct {
	response *transport.HttpResponse
	err      error
}

func NewGrpcHttpProxy(prefix string, serverStream transport.HttpReverseProxy_ReverseProxyServer, logger *log.Logger) (GrpcHttpProxy, error) {
	p, _ := peer.FromContext(serverStream.Context())
	proxy := &grpcHttpProxyImpl{
		prefix:           prefix,
		serverStream:     serverStream,
		requestCh:        make(chan *transport.HttpRequest),
		responseChannels: make(map[int64]chan responseWithError),
		logger:           logger,
		ctx:              serverStream.Context(),
		peerAddr:         p.Addr,
	}

	return proxy, nil
}

// Run - neither grpc.ServerStream#SendMsg nor grpc.ServerStream#RecvMsg are thread-safe.
// We delegate all of those method calls to a single goroutine to avoid any errors with thread safety.
func (g grpcHttpProxyImpl) Run() {
	go func() {
		err := g.sendRequests()
		if err != nil {
			g.logger.Printf("Error sending request to prefix %s: %v", g.prefix, err)
		}
	}()

	go func() {
		err := g.receiveResponses()
		if err != nil {
			g.logger.Printf("Error receiving responses to prefix %s: %v", g.prefix, err)
		}
	}()
}

func (g grpcHttpProxyImpl) SubmitRequest(ctx context.Context, request *transport.HttpRequest) (int64, error) {
	requestId := rand.Int63()

	// Buffer size = 1 lets us not block when writing to the response channel
	g.responseChannels[requestId] = make(chan responseWithError, 1)
	request.RequestId = requestId
	select {
	case g.requestCh <- request:
		return requestId, nil
	case <-ctx.Done():
		return -1, context.Canceled
	}
}

func (g grpcHttpProxyImpl) AwaitResponse(ctx context.Context, requestId int64) (*transport.HttpResponse, error) {
	select {
	case response := <-g.responseChannels[requestId]:
		if response.err != nil {
			return nil, response.err
		}

		delete(g.responseChannels, requestId)

		return response.response, nil
	case <-ctx.Done():
		return nil, context.Canceled
	}
}

func (g grpcHttpProxyImpl) sendRequests() error {
	done := g.Context().Done()

	for {
		select {
		case <-done:
			return nil
		case req := <-g.requestCh:
			g.logger.Printf("Got request %s for %v (%v)\n", req.Path, g.prefix, g.peerAddr)
			err := sendRequestToClient(g.serverStream, req)
			if err != nil {
				return err
			}
		}
	}
}

func (g grpcHttpProxyImpl) receiveResponses() error {
	done := g.Context().Done()
	responseCh := make(chan *transport.HttpResponse)

	go func() {
		for {
			response := transport.HttpResponse{}
			err := g.serverStream.RecvMsg(&response)
			if err == io.EOF || status.Convert(err).Code() == codes.Canceled {
				return
			} else if err != nil {
				g.logger.Printf("Error trying to receive GRPC response: %v\n", err)
				return
			}
			responseCh <- &response
		}
	}()

	for {
		select {
		case <-done:
			return nil
		case response := <-responseCh:
			requestResponseCh := g.responseChannels[response.RequestId]
			if requestResponseCh == nil {
				g.logger.Printf("Nil response channel for prefix (%s) request id (%d)", g.prefix, response.RequestId)
				continue
			}

			requestResponseCh <- responseWithError{
				response: response,
				err:      nil,
			}
		}
	}
}

func sendRequestToClient(server transport.HttpReverseProxy_ReverseProxyServer, request *transport.HttpRequest) error {
	requestResponse := &transport.ReverseProxyResponse_HttpRequest{HttpRequest: request}
	return server.Send(&transport.ReverseProxyResponse{Response: requestResponse})
}
