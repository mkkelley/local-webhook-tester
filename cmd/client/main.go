package main

import (
	"bytes"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"io"
	"local-webhook-tester/transport"
	"net/http"
)

func echo(writer http.ResponseWriter, request *http.Request) {
	b, _ := io.ReadAll(request.Body)
	_, _ = writer.Write([]byte(b))
}

func main() {
	go http.ListenAndServe(":8082", http.HandlerFunc(echo))
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(":3032", opts...)
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
	fmt.Println(response)

	for {
		req := transport.ReverseProxyResponse{}
		err = response.RecvMsg(&req)
		fmt.Println(req.Response)

		switch x := req.Response.(type) {
		case *transport.ReverseProxyResponse_ProxyStartResponse:
			panic("?????")
		case *transport.ReverseProxyResponse_HttpRequest:
			request, err := http.NewRequest(x.HttpRequest.Method, fmt.Sprintf("http://localhost:8082/%s", x.HttpRequest.Path), bytes.NewReader([]byte(x.HttpRequest.Body)))
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
			fmt.Println(string(y))

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
