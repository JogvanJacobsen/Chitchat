package main

import (
	proto "ChitChatServer/grpc"
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
)

type ChitChatServer struct {
	proto.UnimplementedChitChatServer
	mu      sync.Mutex
	clients map[string]chan *proto.ChatMessage
	clock   uint64
}

func main() {
	server := ChitChatServer{clients: make(map[string]chan *proto.ChatMessage)}
	log.Print("ChitChat server has been booted up")

	server.start_server()
	log.Print("ChitChat server has been shut down")
}

func (s *ChitChatServer) start_server() {
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":5050")
	if err != nil {
		log.Fatalf("Did not work")
	}

	proto.RegisterChitChatServer(grpcServer, s)

	err = grpcServer.Serve(listener)

	if err != nil {
		log.Fatalf("Did not work")
	}

}

// handles a client joining the chat server
func (s *ChitChatServer) JoinServer(ctx context.Context, req *proto.JoinServerRequest) (*proto.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If the client is already connected, continue
	if _, exists := s.clients[req.Username]; exists {
		return &proto.Empty{}, nil
	}

	// Create a buffered channel for the new client.
	ch := make(chan *proto.ChatMessage, 32)
	s.clients[req.Username] = ch

	// Increment the Lamport clock.
	s.clock = max(s.clock, req.Timestamp) + 1

	// Send a “user joined the chat” message to all clients in server.
	joinMsg := &proto.ChatMessage{
		Sender:    "Server",
		Message:   fmt.Sprintf("%s joined the Chat", req.Username),
		Timestamp: s.clock,
	}
	for _, clientCh := range s.clients {
		clientCh <- joinMsg
	}
	log.Printf("[Server] Join: user=%s lt=%d", req.Username, s.clock)

	return &proto.Empty{}, nil
}

// handles a client leaving the chat.
func (s *ChitChatServer) LeaveServer(ctx context.Context, req *proto.LeaveServerRequest) (*proto.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find and remove the client’s channel.
	ch, exists := s.clients[req.Username]
	if !exists {
		return &proto.Empty{}, nil
	}
	delete(s.clients, req.Username)
	close(ch)

	// Increment the Lamport clock.
	s.clock = max(s.clock, req.Timestamp) + 1

	// Send a “user left the chat” message to the remaining clients.
	leaveMsg := &proto.ChatMessage{
		Sender:    "Server",
		Message:   fmt.Sprintf("%s left the Chat at logical time %d", req.Username, s.clock),
		Timestamp: s.clock,
	}
	for _, clientCh := range s.clients {
		clientCh <- leaveMsg
	}
	log.Printf("[Server] Leave: user=%s lt=%d", req.Username, s.clock)

	return &proto.Empty{}, nil
}

// PublishMessage sends a user message to all clients.
func (s *ChitChatServer) PublishMessage(ctx context.Context, msg *proto.ChatMessage) (*proto.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Increment the Lamport clock and timestamp the message.
	s.clock = max(s.clock, msg.Timestamp) + 1
	msg.Timestamp = s.clock

	// send the message out to all client channels.
	for _, ch := range s.clients {

		select {
		case ch <- msg:
		default:
		}
	}
	log.Printf("[Server] Message: from=%s lt=%d", msg.Sender, s.clock)

	return &proto.Empty{}, nil
}

// ReceiveMessages notifies messages to a specific client.
func (s *ChitChatServer) ReceiveMessages(req *proto.ReceiveMessagesRequest, stream proto.ChitChat_ReceiveMessagesServer) error {
	s.mu.Lock()
	ch, ok := s.clients[req.Username]
	s.mu.Unlock()
	if !ok {
		// if client isn't in the chat; return
		return nil
	}

	// Read from the client's channel and notify messages back in server.
	for msg := range ch {
		s.mu.Lock()
		s.clock++
		currentTimestamp := s.clock
		msg.Timestamp = currentTimestamp
		s.mu.Unlock()

		if err := stream.Send(msg); err != nil {
			return err
		}
		log.Printf("[Server] Notice: to=%s lt=%d", req.Username, currentTimestamp)
	}
	return nil
}
