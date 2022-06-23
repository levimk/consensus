package node

import (
	"context"
	"fmt"
	"log"
	pb "raft/infrastructure"
	"raft/node/state"
	"raft/types"
	"sync"
	"time"
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
}

func NewRaft(ctx context.Context, id int) *RaftNode {
	raft := &RaftNode{
		Id:            types.NodeId(id),
		State:         state.NewState(),
		peers:         make(map[types.NodeId]*RaftPeer),
		ctx:           ctx,
		electionTimer: newElectionTimer(),
		entryAppended: make(chan bool),
		voteGranted:   make(chan bool),
	}
	return raft
}

func (node *RaftNode) RunClient(abort chan bool) {
	fmt.Println(fmt.Sprintf("\t[%d] starting gRPC server", node.Id))
	for {
		var stateChange state.StateTransition = <-node.State.WatchStateChange
		switch stateChange.To {
		case state.Follower:
			// handleBecomeFollower
			go node.runFollower()
			break
		case state.Candidate:
			// handleBecomeCandidate
			go node.Election()
			break
		case state.Leader:
			// handleBecomeLeader
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
