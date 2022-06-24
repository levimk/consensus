package consensus

// TODO: deadlines: https://grpc.io/blog/deadlines/

import (
	"context"
	"fmt"
	"log"
	"raft/consensus/state"
	pb "raft/infrastructure"
	"raft/types"
)

// #####----- CLIENT -----#####
// RaftNode.RequestVoteBroadcast: calls RequestVoteRPC on each peer
func (node *RaftNode) RequestVoteBroadcast() {
	if node.State.GetServerState() == state.Leader {
		msg := fmt.Sprintf("RequestVoteError: [%d] tried to call election but is a %s", node.Id, node.State.GetServerState().String())
		log.Fatalln(msg)
	}
	fmt.Println(fmt.Sprintf("\t\t[%d] requesting votes", node.Id))
	submitVote := make(chan bool)
	votes := 1 // Node automatically votes for itself when it calls an election
	for peerId, peer := range node.peers {
		go func(peerId types.NodeId, peer *RaftPeer) {
			term := node.State.CurrentTerm
			candidateId := node.Id
			lastLogIndex := node.State.GetLastLogIndex()
			var lastLogEntry types.Entry
			if lastLogIndex == -1 {
				lastLogEntry = types.Entry{}
			} else {
				var getEntryErr error
				lastLogEntry, getEntryErr = node.State.GetEntryAt(int(lastLogIndex))
				if getEntryErr != nil {
					log.Println(getEntryErr)
				}
			}
			// TODO: handle reply and error
			reply, err := peer.RequestVote(
				node.ctx,
				term,
				candidateId,
				lastLogIndex,
				lastLogEntry.Term,
			)
			if err != nil {
				// log.Println(err)
				return
			}
			if reply.VoteGranted {
				submitVote <- reply.VoteGranted
			} else {
				node.mtx.Lock()
				if node.State.CurrentTerm < types.Term(reply.CurrentTerm) {
					// I'm out of date - become a follower and update my term
					node.State.CurrentTerm = types.Term(reply.CurrentTerm)
					if node.State.GetServerState() == state.Candidate {
						// TODO @byzantine - not safe => can get tricked by opponent
						candidateToFollower, stateChangeErr := node.State.DiscoverCurrentLeaderOrNewTerm()
						if stateChangeErr != nil {
							log.Fatalln(stateChangeErr)
						}
						node.State.SetServerState(candidateToFollower)
						node.State.WatchStateChange <- state.StateTransition{From: state.Candidate, To: state.Follower}
					}
				}
				node.mtx.Unlock()
			}
		}(peerId, peer)
	}
	for {
		select {
		case <-node.electionTimer.EndCurrentElection:
			// Wait until the election timer is finished (either election timeout or stopped)
			return
		case receivedVote := <-submitVote:
			node.mtx.Lock()
			if receivedVote {
				votes++
			}
			wonElection := votes > ((len(node.peers) + 1) / 2)
			stillCandidate := node.State.GetServerState() == state.Candidate
			// only trigger the state change CANDIDATE -> LEADER
			if wonElection && stillCandidate {
				// TODO: improve this mess...
				becomeLeader, err := node.State.ReceivedMajorityVote()
				if err != nil {
					log.Fatalln(err)
				}
				node.State.SetServerState(becomeLeader)
				node.State.WatchStateChange <- state.StateTransition{From: state.Candidate, To: state.Leader}
				node.electionTimer.Stop()
				node.mtx.Unlock()
				return
			}
			node.mtx.Unlock()
		}
	}
}

// RaftPeer.RequestVote: call on a peer to execute a RequestVote RPC
func (peer *RaftPeer) RequestVote(
	ctx context.Context,
	term types.Term,
	candidateId types.NodeId,
	lastLogIndex types.LogIndex,
	lastLogTerm types.Term) (*pb.VoteReply, error) {
	msg := pb.VoteRequest{
		Term:         int32(term),
		CandidateId:  int32(candidateId),
		LastLogIndex: int32(lastLogIndex),
		LastLogTerm:  int32(lastLogTerm),
	}
	reply, err := peer.Client.RequestVote(ctx, &msg)
	return reply, err
}

// #####----- SERVER -----#####
// RaftNode.RequestVote: this is executed on the node when the leader calls RaftPeer.RequestVote
func (server *RaftNode) RequestVote(ctx context.Context, message *pb.VoteRequest) (*pb.VoteReply, error) {
	server.mtx.Lock()
	defer server.mtx.Unlock()
	currentTerm := int32(server.State.GetCurrentTerm())
	reply := pb.VoteReply{
		CurrentTerm: currentTerm,
		VoteGranted: false, // default
	}
	// Ongaro & Ousterhout pg308.
	// RequestVote RPC | Receiver implementation (1)
	if message.Term < currentTerm {
		reply.VoteGranted = false
	} else {
		// RequestVote RPC | Receiver implementation (2)
		if currentTerm < message.Term {
			// reset my vote if I haven't voted in this term before
			server.State.VotedFor = state.NO_VOTE
			server.State.CurrentTerm = types.Term(message.Term)

			// I'm a LEADER but the requester has a higher state - become follower
			if server.State.GetServerState() == state.Leader {
				// TODO @byzantine - not safe => can get bullied by opponent
				leaderToFollower, stateChangeErr := server.State.DiscoverServerWithHigherTerm()
				if stateChangeErr != nil {
					log.Fatalln(stateChangeErr)
				}
				server.State.SetServerState(leaderToFollower)
				server.State.WatchStateChange <- state.StateTransition{From: state.Leader, To: state.Follower}
			}
			// I'm a CANDIDATE but the requester has a higher state - become follower
			if server.State.GetServerState() == state.Candidate {
				// TODO @byzantine - not safe => can get bullied by opponent
				candidateToFollower, stateChangeErr := server.State.DiscoverCurrentLeaderOrNewTerm()
				if stateChangeErr != nil {
					log.Fatalln(stateChangeErr)
				}
				server.State.SetServerState(candidateToFollower)
				server.State.WatchStateChange <- state.StateTransition{From: state.Candidate, To: state.Follower}
			}
			// @byzantine
			// TODO: is this a possible BFT exploit?
			// TODO: exploit constantly increasing message.Term!
		}

		hasVoted := server.State.VotedFor != state.NO_VOTE
		if !hasVoted {
			// set my vote to this candidate if I haven't voted yet
			server.State.VotedFor = types.NodeId(message.CandidateId)
		}

		votedForCandidate := server.State.VotedFor == types.NodeId(message.CandidateId)
		// TODO: remove? @tidy @refactor
		// if votedForCandidate {
		// 	// pg 308:
		// 	//   If election timeout elapses without receiving AppendEntries
		// 	//   RPC from current leader or granting vote to candidate: convert to candidate
		// 	server.voteGranted <- true
		// }

		candidateLogIsUpToDate := message.LastLogIndex >= int32(server.State.GetLastLogIndex())
		if (!hasVoted || votedForCandidate) && candidateLogIsUpToDate {
			// @byzantine
			// Note: modification from original - !hasVoted (original uses hasVoted)
			//       Why? Stop Byzantine attack of getting multiple votes from one peer
			reply.VoteGranted = true
			server.voteGranted <- true
		}
	}
	return &reply, nil
}
