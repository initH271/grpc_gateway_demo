package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	helloworldpb "grpcgateway/api/proto/helloworld/v1"
)

type server struct {
	helloworldpb.UnimplementedGreeterServer
}

func NewServer() *server {
	return &server{}
}

func (s *server) SayHello(ctx context.Context, in *helloworldpb.HelloRequest) (*helloworldpb.HelloReply, error) {
	return &helloworldpb.HelloReply{Message: in.Name + " world"}, nil
}

func runGRPCServer(wg *sync.WaitGroup) {
	defer wg.Done()
	// Create a listener on TCP port
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	// Create a gRPC server object
	s := grpc.NewServer()
	// Attach the Greeter service to the server
	helloworldpb.RegisterGreeterServer(s, &server{})

	// Enable reflection to allow clients query the server's services.
	reflection.Register(s)
	// Serve gRPC Server
	log.Println("Serving gRPC on 0.0.0.0:8080")
	log.Fatal(s.Serve(lis))
}

func runRESTServer(wg *sync.WaitGroup) {
	defer wg.Done()

	ctx := context.Background()
	mux := runtime.NewServeMux()

	conn, err := grpc.DialContext(ctx, "localhost:8080", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	if err := helloworldpb.RegisterGreeterHandler(ctx, mux, conn); err != nil {
		log.Fatalf("Failed to register handler: %v", err)
	}

	log.Println("Started a gRPC gateway server on 0.0.0.0:8081")
	log.Fatalln(http.ListenAndServe(":8081", mux))
}

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	go runGRPCServer(&wg)
	go runRESTServer(&wg)

	wg.Wait()
}
