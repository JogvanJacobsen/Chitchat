package main

import (
	protobuf "ChitChatServer/grpc"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// setup server
	var clock uint64 = 0
	serverAddr := "localhost:5050"
	user := ""
	if len(os.Args) > 1 {
		user = os.Args[1]
	} else if user == "" {
		log.Fatalf("No username found")
	}

	// connect client to server
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	c := protobuf.NewChitChatClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// client joins server and increments logical time clock
	clock++
	if _, err := c.JoinServer(ctx, &protobuf.JoinServerRequest{Username: user, Timestamp: clock}); err != nil {
		log.Fatalf("join: %v", err)
	}
	log.Printf("joined as %s", user)

	// client receives message and increments logical time clock in the background
	stream, err := c.ReceiveMessages(ctx, &protobuf.ReceiveMessagesRequest{Username: user})
	if err != nil {
		log.Fatalf("receive: %v", err)
	}
	go func() {
		for {
			m, err := stream.Recv()
			if err != nil {
				log.Printf("recv closed: %v", err)
				return
			}
			clock = max(clock, m.Timestamp) + 1
			fmt.Printf("%s: %s, lt=%d\n", m.Sender, m.Message, clock)
		}
	}()

	// handler to leave server (Ctrl-R)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		clock++
		_, _ = c.LeaveServer(context.Background(), &protobuf.LeaveServerRequest{Username: user, Timestamp: clock})
		os.Exit(0)
	}()

	// lets user know how to interact in the server,
	// and publishes messages with incremented logical time clock
	sc := bufio.NewScanner(os.Stdin)
	fmt.Println("*** Type and press enter to send chat -- press Ctrl-C to leave server ***")
	for sc.Scan() {
		text := sc.Text()
		// if the message is empty or too long, ignore it
		if text == "" || len(text) > 128 {
			continue
		}
		clock++
		_, err := c.PublishMessage(ctx, &protobuf.ChatMessage{Sender: user, Message: text, Timestamp: clock})
		if err != nil {
			log.Printf("Message: %v", err)
		}
	}
}
