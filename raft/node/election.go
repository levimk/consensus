package node

import (
	"fmt"
	"log"
	"raft/node/state"
)

func (node *RaftNode) Election() {
	if node.State.GetServerState() == state.Leader {
		msg := fmt.Sprintf("StartElectionError: [%d] tried to call election but is a %s", node.Id, node.State.GetServerState().String())
		log.Fatalln(msg)
	}
	// Start the election timer
	node.startElection()
	for {
		select {
		case lapsed := <-node.electionTimer.Lapsed:
			if lapsed && node.State.GetServerState() == state.Candidate {
				node.startElection()
			}
		case <-node.electionTimer.NoMoreElections:
			return
		}
	}
}

func (node *RaftNode) startElection() {
	node.mtx.Lock()
	node.State.CurrentTerm = node.State.CurrentTerm + 1
	node.State.VotedFor = node.Id
	go node.electionTimer.Start()
	node.mtx.Unlock()
	go node.RequestVoteBroadcast()
}
