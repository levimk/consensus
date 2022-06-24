package consensus

import (
	"context"
	"fmt"
	"log"
	"net"
	"raft/consensus/config"
	"raft/consensus/state"
	pb "raft/infrastructure"
	"raft/types"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
)

const HEARTBEAT int = 50
const HEARTBEAT_DURATION time.Duration = time.Millisecond * time.Duration(HEARTBEAT)

type RaftNode struct {
	pb.UnimplementedRaftServer
	Id            types.NodeId
	State         state.NodeState
	peers         map[types.NodeId]*RaftPeer
	mtx           sync.Mutex
	ctx           context.Context
	electionTimer ElectionTimer
	entryAppended chan bool
	voteGranted   chan bool
	Config        config.RaftConfig
}

func NewRaft(ctx context.Context) *RaftNode {
	id := types.NodeId(ctx.Value("id").(int))
	configPath := ctx.Value("configPath").(string)
	config, err := config.NewConfig(id, configPath)
	if err != nil {
		log.Fatalln(err)
	}
	raft := &RaftNode{
		Id:            types.NodeId(id),
		State:         state.NewState(),
		peers:         make(map[types.NodeId]*RaftPeer),
		ctx:           ctx,
		electionTimer: newElectionTimer(),
		entryAppended: make(chan bool),
		voteGranted:   make(chan bool),
		Config:        config,
	}
	return raft
}

func (self *RaftNode) Run(abort chan bool) {
	id := self.Config.Id
	hostname := self.Config.Server.Hostname
	port := self.Config.Server.Port
	protocol := self.Config.Server.Communication.Protocol
	// Setup LISTEN
	lis, err := net.Listen(protocol, fmt.Sprintf("%s:%d", hostname, port))
	if err != nil {
		log.Println(fmt.Sprintf("%s failed to listen: %v", hostname, err))
		abort <- true
	} else {
		fmt.Println(fmt.Sprintf("Raft[%d] listening on %s:%d", id, hostname, port))
	}
	// Create gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterRaftServer(grpcServer, self)
	fmt.Println(fmt.Sprintf("\t[%d] starting gRPC server", id))

	// Start the RPC client and server
	go self.RunServerState(abort)
	serverAbort := make(chan bool)
	go func(abort chan bool) {
		rpcServerError := grpcServer.Serve(lis)
		if rpcServerError != nil {
			abort <- true
		}
	}(serverAbort)
	for _, peer := range self.Config.Server.Peers {
		go func(peer string) {
			peerDetails := strings.Split(peer, ":")
			if len(peerDetails) != 3 {
				log.Fatalln("Invalid peer details:", peerDetails)
			}

			peerId, idErr := strconv.Atoi(peerDetails[0])
			if idErr != nil {
				log.Fatalln(idErr)
			}
			peerHostname := peerDetails[1]
			peerPort, portErr := strconv.Atoi(peerDetails[2])
			if portErr != nil {
				log.Fatalln(portErr)
			}
			self.AddPeer(types.NodeId(peerId), peerHostname, peerPort)
		}(peer)
	}
	<-serverAbort
	abort <- true
}

// Returns true of this node is the leader, false otherwise to signal
// that the caller should find a way to forward their request to the leader

// Original version: takes generic object handling multiple commands => delegate to SubmitBatch
// func (self *RaftNode) Submit(object interface{}) bool {
// 	log.Printf("Submit: %v\n", object)
// 	return self.SubmitBatch([]interface{}{object})
// }
func (self *RaftNode) Submit(command types.Command) error {
	err := self.State.AppendEntry(self.State.GetNextLogIndex(), self.State.CurrentTerm, command)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(fmt.Sprintf("[%d] LOG APPEND: Client(%s) %s=%s", self.Id, command.ClientId, command.Command, command.Value))
	}
	return err
}

func (node *RaftNode) RunServerState(abort chan bool) {
	fmt.Println(fmt.Sprintf("\t[%d] starting server state", node.Id))
	fmt.Println(fmt.Sprintf("\t\t[%d] begin as FOLLOWER", node.Id))
	go node.runFollower()
	for {
		var stateChange state.StateTransition = <-node.State.WatchStateChange
		switch stateChange.To {
		case state.Follower:
			// handleBecomeFollower
			fmt.Println(fmt.Sprintf("\t[%d] becoming FOLLOWER", node.Id))
			go node.runFollower()
			break
		case state.Candidate:
			// handleBecomeCandidate
			fmt.Println(fmt.Sprintf("\t[%d] becoming CANDIDATE", node.Id))
			go node.Election()
			break
		case state.Leader:
			// handleBecomeLeader
			fmt.Println(fmt.Sprintf("\t[%d] becoming LEADER - term=%d", node.Id, node.State.CurrentTerm))
			go node.runLeader() // TODO: initial vs repeating
			break
		case state.INVALID:
			abort <- true
			return
		}
	}
}

func (node *RaftNode) runLeader() {
	heartBeat := make(chan bool)
	stopTimer := make(chan bool)
	node.State.ReinitialiseLeaderState()
	go startTimer(HEARTBEAT_DURATION, stopTimer, heartBeat)
	go node.AppendEntriesBroadcast()
	for {
		select {
		// pg 308:
		case <-node.entryAppended:
			// a new leader has told me to append entries so I need to stop
			return
		case <-node.voteGranted:
			// possible late messages coming from an election I already won
			// this clause is so the channel is emptied...
			// TODO: is this actual how channels work?
			break
		case <-heartBeat:
			go startTimer(HEARTBEAT_DURATION, stopTimer, heartBeat)
			go node.AppendEntriesBroadcast()
			break
		}
	}
}

func (node *RaftNode) runFollower() {
	var expectHeartbeat time.Duration = time.Duration(time.Second * 1) // TODO: refactor
	timedOut := make(chan bool)
	stopTimer := make(chan bool)
	go startTimer(expectHeartbeat, stopTimer, timedOut)
	for {
		select {
		// pg 308:
		// If election timeout elapses without receiving AppendEntries
		// RPC from current leader or granting vote to candidate: convert to candidate
		case <-node.entryAppended:
			stopTimer <- true
			go startTimer(expectHeartbeat, stopTimer, timedOut)
			break
		case <-node.voteGranted:
			stopTimer <- true
			go startTimer(expectHeartbeat, stopTimer, timedOut)
			break
		case <-timedOut:
			node.mtx.Lock()
			followerToCandidate, err := node.State.TimeoutAndStartElection()
			if err != nil {
				log.Println(err)
			}
			node.State.SetServerState(followerToCandidate)
			node.State.WatchStateChange <- state.StateTransition{From: state.Follower, To: state.Candidate}
			node.mtx.Unlock()
			return
		}
	}
}

func startTimer(duration time.Duration, stop chan bool, timedOut chan bool) {
	for {
		select {
		case <-stop:
			return
		case <-time.After(duration):
			timedOut <- true
		}
	}
}
