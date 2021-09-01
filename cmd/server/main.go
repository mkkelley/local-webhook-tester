package main

import (
	"fmt"
	"google.golang.org/grpc"
	"local-webhook-tester/proxy"
	"local-webhook-tester/transport"
	"net"
)

func main() {
	config := &proxy.ServerConfig{
		BaseUrl:  "http://localhost:3031/",
		HttpPort: "3031",
		GrpcPort: "3032",
	}

	prx := proxy.NewReverseProxy(config)
	go proxy.RunHttpServer(config, &prx)

	server := grpc.NewServer()
	server.RegisterService(&transport.HttpReverseProxy_ServiceDesc, prx)
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%s", config.GrpcPort))
	fmt.Println("done")
	err := server.Serve(listener)
	if err != nil {
		panic(err)
	}
	fmt.Println("really done")
}
