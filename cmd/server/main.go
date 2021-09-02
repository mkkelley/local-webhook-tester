package main

import (
	"crypto/tls"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

	var opts []grpc.ServerOption
	if config.UseTls {
		cert, err := tls.LoadX509KeyPair("server-cert.pem", "server-key.pem")
		if err != nil {
			panic(err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.NoClientCert,
		}

		transportCredentials := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.Creds(transportCredentials))
	}
	server := grpc.NewServer(opts...)
	server.RegisterService(&transport.HttpReverseProxy_ServiceDesc, prx)
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%s", config.GrpcPort))
	fmt.Printf("GRPC listening on %s\n", config.GrpcPort)
	err = server.Serve(listener)
	if err != nil {
		panic(err)
	}
	fmt.Println("really done")
}
