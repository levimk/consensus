package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"raft/consensus"
)

var (
	id          = flag.Int("id", -1, "Node id")
	serveClient = flag.Bool("serveClient", false, "Delegate this server as the client-facing server")
	raftConfig  = flag.String("config", "", "Path to directory containing config file")
)

func main() {
	fmt.Println("Raft")
	flag.Parse()
	if *id == -1 {
		log.Fatalln("Must provide node id >= 0 using `-id` flag")
	}
	if *raftConfig == "" {
		log.Fatalln("Must provide Raft config file (yaml) `-config` flag")
	}
	raftCtx := context.Background()
	raftCtx = context.WithValue(raftCtx, "id", *id)
	raftCtx = context.WithValue(raftCtx, "configPath", *raftConfig)
	raft := consensus.NewRaft(raftCtx)

	initiateShutDown := make(chan bool)
	shutDownSystem := make(chan bool)

	// Listen for connections and communication
	go raft.Run(initiateShutDown)

	// handle shut down
	go func() {
		<-initiateShutDown
		shutDownSystem <- true
	}()

	clientServer := NewClientServer(raft)

	if *serveClient {
		go clientServer.Run()
	}

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
