package main

import (
	"fmt"
	"google.golang.org/grpc"
	"local-webhook-tester/proxy"
	"local-webhook-tester/transport"
	"net"
)

func main() {
	config, err := proxy.ReadConfig()
	if err != nil {
		panic(err)
	}

	prx := proxy.NewReverseProxy(config)
	go func() {
		fmt.Printf("HTTP will listen on %s\n", config.HttpPort)
		err = proxy.RunHttpServer(config, &prx)
		if err != nil {
			panic(err)
		}
	}()

	server := grpc.NewServer()
	server.RegisterService(&transport.HttpReverseProxy_ServiceDesc, prx)
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%s", config.GrpcPort))
	fmt.Printf("GRPC listening on %s\n", config.GrpcPort)
	err = server.Serve(listener)
	if err != nil {
		panic(err)
	}
	fmt.Println("really done")
}
