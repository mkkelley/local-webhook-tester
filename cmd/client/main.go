package main

import (
	"bytes"
	"context"
	"flag"
	"google.golang.org/grpc"
	"io"
	"local-webhook-tester/transport"
	"log"
	"net/http"
	"net/url"
	"os"
)

func echo(writer http.ResponseWriter, request *http.Request) {
	b, _ := io.ReadAll(request.Body)
	_, _ = writer.Write([]byte(b))
}

func main() {
	allowInsecure := flag.Bool("insecure", false, "Allow the GRPC client to connect over HTTP")
	proxyServer := flag.String("proxy-server", "localhost:3032", "GRPC server URL")
	server := flag.String("server", "http://localhost:8082", "URL to which to proxy requests")
	runTestServer := flag.Bool("run-test-server", false, "")

	flag.Parse()

	if *runTestServer {
		go http.ListenAndServe(":8082", http.HandlerFunc(echo))
	}

	logger := log.New(os.Stdout, "[client] ", log.LstdFlags)

	var opts []grpc.DialOption
	if *allowInsecure {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(*proxyServer, opts...)
	if err != nil {
		panic(err)
	}
	client := transport.NewHttpReverseProxyClient(conn)
	response, err := client.ReverseProxy(context.Background())
	if err != nil {
		panic(err)
	}
	message := transport.ReverseProxyResponse{}
	err = response.RecvMsg(&message)
	if err != nil {
		panic(err)
	}
	sr := message.Response.(*transport.ReverseProxyResponse_ProxyStartResponse)
	logger.Printf("Set http calls to point to %s", sr.ProxyStartResponse.BaseUrl)

	for {
		req := transport.ReverseProxyResponse{}
		err = response.RecvMsg(&req)
		if err != nil {
			panic(err)
		}
		logger.Println(req.Response)

		switch x := req.Response.(type) {
		case *transport.ReverseProxyResponse_ProxyStartResponse:
			panic("?????")
		case *transport.ReverseProxyResponse_HttpRequest:
			localUrl, err := url.Parse(*server)
			if err != nil {
				panic(err)
			}
			localUrl.Path = x.HttpRequest.Path

			request, err := http.NewRequest(x.HttpRequest.Method, localUrl.String(), bytes.NewReader([]byte(x.HttpRequest.Body)))
			if err != nil {
				panic(err)
			}
			re, err := http.DefaultClient.Do(request)
			if err != nil {
				panic(err)
			}
			y, err := io.ReadAll(re.Body)
			if err != nil {
				panic(err)
			}

			transportResponse := transport.HttpResponse{
				ResponseCode: int32(re.StatusCode),
				Body:         string(y),
				Headers:      []string{},
			}
			err = response.Send(&transportResponse)
			if err != nil {
				panic(err)
			}
		}
	}
}
