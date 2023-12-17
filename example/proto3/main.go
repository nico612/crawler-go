package main

import (
	"context"
	"github.com/nico612/crawler-go/example/proto3/v1"
	"google.golang.org/grpc"
	"log"
	"net"
)

type Greeter struct {
	v1.UnimplementedGreeterServer
}

// Hello 实现grpc服务接口
func (g *Greeter) Hello(ctx context.Context, request *v1.Request) (rsp *v1.Response, err error) {
	rsp.Greeting = "hello " + request.Name
	return
}

func (g *Greeter) mustEmbedUnimplementedGreeterServer() {
}

func main() {
	println("gRPC server tutorial in Go")
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()
	v1.RegisterGreeterServer(s, &Greeter{})

	if err := s.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
