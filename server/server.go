package main

import (
	proto "ChitChatServer/grpc"
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
)

type ChitChat_Server struct {
	proto.UnimplementedChitChatClientsServer
	clients []string
}

func main() {
	server := ChitChat_Server{clients: []string{}}

	log.Print("Server has been started")

	server.clients = append(server.clients, "John W. Lennon")

	server.start_server()
	log.Print("Server has been terminated")
}

func (s *ChitChat_Server) GetClients(ctx context.Context, in *proto.Empty) (*proto.Clients, error) {
	return &proto.Clients{Clients: s.clients}, nil
}

func (s *ChitChat_Server) start_server() {
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":5050")
	if err != nil {
		log.Print(err)
		log.Fatalf("Did not work")
	}

	proto.RegisterChitChatClientsServer(grpcServer, s)

	err = grpcServer.Serve(listener)

	if err != nil {
		log.Fatalf("Did not work")
	}

}
