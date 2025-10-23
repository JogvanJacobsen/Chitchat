package main

import (
	proto "ChitChatServer/grpc"
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:5050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Not working")
	}

	client := proto.NewChitChatClientsServer(conn)

	clients, err := client.GetClients(context.Background(), &proto.Empty{})
	if err != nil {
		log.Print(err)
		log.Fatalf("Not working")
	}

	for _, client := range clients.Clients {
		println(" - " + client)
	}
}
