package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io"
	"local-webhook-tester/transport"
	"local-webhook-tester/util"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func echo(writer http.ResponseWriter, request *http.Request) {
	b, _ := io.ReadAll(request.Body)
	_, _ = writer.Write([]byte(b))
}

func main() {
	allowPlaintext := flag.Bool("plaintext", false, "Allow the GRPC client to connect over HTTP")
	proxyServer := flag.String("proxy-server", "proxy-conn.minthe.net:443", "GRPC server URL")
	server := flag.String("server", "http://localhost:8082", "URL to which to proxy requests")
	runTestServer := flag.Bool("run-test-server", false, "")
	hostHeader := flag.String("host", "", "Set a host header for local reeuests")

	flag.Parse()

	if *runTestServer {
		go func() {
			_ = http.ListenAndServe(":8082", http.HandlerFunc(echo))
		}()
	}

	logger := log.New(os.Stdout, "[client] ", log.LstdFlags)

	var opts []grpc.DialOption
	if *allowPlaintext {
		opts = append(opts, grpc.WithInsecure())
	} else {
		tlsConfig := tls.Config{}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tlsConfig)))
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
	logger.Printf("Set http calls to point to %s/", sr.ProxyStartResponse.BaseUrl)

	for {
		req := transport.ReverseProxyResponse{}
		err = response.RecvMsg(&req)
		if err != nil {
			logger.Fatal(err)
		}
		logger.Println("Got request: ", req.Response)

		switch x := req.Response.(type) {
		case *transport.ReverseProxyResponse_ProxyStartResponse:
			logger.Fatal("?????")
		case *transport.ReverseProxyResponse_HttpRequest:
			localUrl, err := url.Parse(*server)
			if err != nil {
				logger.Fatal(err)
			}
			localUrl.Path = x.HttpRequest.Path

			localRequest, err := http.NewRequest(x.HttpRequest.Method, localUrl.String(), bytes.NewReader([]byte(x.HttpRequest.Body)))
			if err != nil {
				logger.Fatal(err)
			}

			addRequestHeaders(hostHeader, localRequest, x)

			localResponse, err := http.DefaultClient.Do(localRequest)
			if err != nil {
				logger.Fatal(err)

			}
			responseBody, err := io.ReadAll(localResponse.Body)
			if err != nil {
				logger.Fatal(err)
			}

			transportResponse := &transport.HttpResponse{
				ResponseCode: int32(localResponse.StatusCode),
				Body:         string(responseBody),
				Headers:      util.SerializeHeader(localResponse.Header),
			}
			logger.Println("Sending response: ", transportResponse)
			err = response.Send(transportResponse)
			if err != nil {
				logger.Fatal(err)
			}
		}
	}
}

func addRequestHeaders(hostHeader *string, request *http.Request, x *transport.ReverseProxyResponse_HttpRequest) {
	if *hostHeader != "" {
		request.Header.Set("Host", *hostHeader)
	}
	for _, header := range x.HttpRequest.Headers {
		split := strings.Split(header, ":")
		key := split[0]
		val := split[1][1:]

		request.Header.Add(key, val)
	}
}
