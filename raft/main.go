package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	pb "raft/infrastructure"
	"raft/node"

	"google.golang.org/grpc"
)

const CONNECTION_PROTOCOL string = "tcp"

var (
	id       = flag.Int("id", 1, "Node id")
	hostname = flag.String("hostname", "localhost", "The server hostname")
	port     = flag.Int("port", 8080, "The server port")
)

func main() {
	fmt.Println("Raft")
	flag.Parse()
	lis, err := net.Listen(CONNECTION_PROTOCOL, fmt.Sprintf("%s:%d", *hostname, *port))
	if err != nil {
		log.Fatalf("%s failed to listen: %v", *hostname, err)
	} else {
		config := fmt.Sprintf("\tNode[%d] listening on %s:%d", *id, *hostname, *port)
		fmt.Println(config)
	}
	grpcServer := grpc.NewServer()
	mainCtx := context.Background()
	raft := node.NewRaft(mainCtx, *id)

	// Listen for connections and communication
	initiateShutDown := make(chan bool)
	shutDownSystem := make(chan bool)
	go func() {
		fmt.Println(fmt.Sprintf("\t[%d] starting gRPC server", *id))
		pb.RegisterRaftServer(grpcServer, raft)
		grpcServer.Serve(lis)
		initiateShutDown <- true
	}()

	// for peer in peers : connect to peer
	go raft.RunClient(initiateShutDown)

	// handle shut down
	go func() {
		<-initiateShutDown
		shutDownSystem <- true
	}()

	<-shutDownSystem
}

// TODO:
// - [ ] pg308: Rules for Servers
// - [ ] Add peers
// - [ ] Commandline arg to add peer
// - [ ] Dolov-Strong: bootstrap internal key registry
// - [ ] Docker
// - [ ] Replace to Order Book's raft layer with this code
// - [ ] Order Book user/client connect to all Raft nodes (for pub key broadcast)
