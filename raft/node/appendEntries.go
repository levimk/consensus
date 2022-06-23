package node

// TODO: deadlines: https://grpc.io/blog/deadlines/

import (
	"context"
	"errors"
	"fmt"
	"log"
	pb "raft/infrastructure"
	"raft/node/state"
	"raft/types"
	rt "raft/types"
	"sync"
)

// #####----- CLIENT -----#####
// RaftNode.AppendEntriesBroadcast: calls AppendEntriesRPC on each peer
func (node *RaftNode) AppendEntriesBroadcast() {
	if node.State.GetServerState() != state.Leader {
		msg := fmt.Sprintf("AppendEntriesError: [%d] tried to broadcast an AppendEntriesRPC but is a %s", node.Id, node.State.GetServerState().String())
		log.Fatalln(msg)
	}
	var wg sync.WaitGroup
	for peerId, peer := range node.peers {
		wg.Add(1)
		go func(peerId types.NodeId, peer *RaftPeer) {
			defer wg.Done()
			node.mtx.Lock()
			prevLogIndex, _ := node.State.GetPrevLogIndexForPeer(peerId)
			prevLogTerm, _ := node.State.GetPrevLogTermForPeer(peerId)
			var entries types.Entries
			if !node.State.HasSentInitialHeartbeat() {
				entries = types.Entries{}
				node.State.MarkHasSentInitialHeartbeat()
			} else {
				entries, _ = node.State.GetEntriesSince(prevLogIndex)
			}
			// TODO: handle reply and error
			lastLogIndex := len(node.State.Log) - 1
			node.mtx.Unlock()
			if lastLogIndex > int(prevLogIndex) {
				reply, err := peer.AppendEntries(
					node.ctx,
					node.State.GetCurrentTerm(),
					node.Id,
					prevLogIndex,
					prevLogTerm,
					entries,
					node.State.GetCommitIndex(),
				)
				if err != nil {
					log.Println(err)
				}
				if reply.Success {
					node.mtx.Lock()
					node.State.UpdateNextIndex(peerId, types.LogIndex(lastLogIndex+1))
					node.State.UpdateMatchIndex(peerId, types.LogIndex(lastLogIndex+1))
					node.mtx.Unlock()
				} else {
					node.mtx.Lock()
					if rt.Term(reply.Term) < node.State.CurrentTerm {
						// Their nextIndex still doesn't match - decrement their next index
						node.State.DecrementNextIndex(peerId)
					} else if rt.Term(reply.Term) > node.State.CurrentTerm {
						// I'm out of date - revert to follower
						node.State.CurrentTerm = rt.Term(reply.Term)
						leaderToFollower, err := node.State.DiscoverServerWithHigherTerm()
						if err != nil {
							log.Println(err)
						}
						node.State.SetServerState(leaderToFollower)
						node.State.WatchStateChange <- state.StateTransition{From: state.Leader, To: state.Follower}
					}
					node.mtx.Unlock()
				}
			}
		}(peerId, peer)
	}
	wg.Wait()
}

// RaftPeer.AppendEntries: call on a peer to execute an AppendEntries RPC
func (peer *RaftPeer) AppendEntries(
	ctx context.Context,
	term rt.Term,
	leaderId rt.NodeId,
	prevLogIndex rt.LogIndex,
	prevLogTerm rt.Term,
	entries rt.Entries,
	leaderCommit rt.CommitIndex) (*pb.AppendEntriesReply, error) {
	// Split entries into component parts for marshalling
	terms := make([]int32, len(entries))
	indexes := make([]int32, len(entries))
	commands := make([]string, len(entries))
	for _, entry := range entries {
		terms = append(terms, int32(entry.Term))
		indexes = append(indexes, int32(entry.Index))
		commands = append(commands, string(entry.Command))
	}
	msg := pb.AppendEntriesRequest{
		Term:         int32(term),
		LeaderId:     int32(leaderId),
		PrevLogIndex: int32(prevLogIndex),
		PrevLogTerm:  int32(prevLogTerm),
		Terms:        terms,
		Indexes:      indexes,
		Commands:     commands,
		LeaderCommit: int32(leaderCommit),
	}
	reply, err := peer.Client.AppendEntries(ctx, &msg)
	return reply, err
}

// #####----- SERVER -----#####
// RaftNode.AppendEntries: this is executed on a node when the leader calls RaftPeer.AppendEntries
func (server *RaftNode) AppendEntries(
	ctx context.Context,
	message *pb.AppendEntriesRequest) (*pb.AppendEntriesReply, error) {
	server.mtx.Lock()
	defer server.mtx.Unlock()

	var currentTerm int32 = int32(server.State.CurrentTerm)

	// Ongaro & Ousterhout pg308.
	// AppendEntries RPC | Receiver implementation (1): §5.1
	if message.Term < currentTerm {
		errMsg := fmt.Sprintf("AppendEntriesServerError: leader [%d]'s term %d < my [%d] current term %d",
			message.LeaderId, message.Term, server.Id, server.State.CurrentTerm)
		error := errors.New(errMsg)
		return &pb.AppendEntriesReply{Term: currentTerm, Success: false}, error
	}

	// PrevLogIndex index is out of range
	if message.PrevLogIndex < 0 || message.PrevLogIndex >= int32(len(server.State.Log)) {
		errMsg := fmt.Sprintf("AppendEntriesServerError: [%d] received out of range log index %d from [%d]",
			server.Id, message.PrevLogIndex, message.LeaderId)
		error := errors.New(errMsg)
		return &pb.AppendEntriesReply{Term: currentTerm, Success: false}, error
	}

	// Get the entry at PrevLogIndex
	entry := server.State.Log[message.PrevLogIndex]

	// Ongaro & Ousterhout pg308.
	// AppendEntries RPC | Receiver implementation (2): §5.2
	if entry.Term != types.Term(message.PrevLogTerm) {
		// TODO: guard for entry.LogIndex != PrevLogIndex?
		errMsg := fmt.Sprintf("AppendEntriesServerError: [%d] has incorrect term=%d at log index %d (should be %d)",
			server.Id, entry.Term, message.PrevLogIndex, message.PrevLogTerm)
		error := errors.New(errMsg)
		// Ongaro & Ousterhout pg308.
		// AppendEntries RPC | Receiver implementation (3)
		server.State.DeleteEntriesFromIndexOnwards(entry.Index)
		return &pb.AppendEntriesReply{Term: currentTerm, Success: false}, error
	}

	// notify other goroutines of appending entry
	// TODO: double check reasoning...
	server.entryAppended <- true

	if currentTerm < message.Term {
		// "Rules for all servers" - dot point 2, page 308
		server.State.CurrentTerm = types.Term(message.Term)
		server.State.VotedFor = state.NO_VOTE
		// TODO @byzantine ???

		// Change state (cancenl candidacy or resign from leadership)
		if server.State.GetServerState() == state.Candidate {
			candidateToFollower, errC2F := server.State.DiscoverCurrentLeaderOrNewTerm()
			if errC2F == nil {
				server.State.SetServerState(candidateToFollower)
				server.State.WatchStateChange <- state.StateTransition{From: state.Candidate, To: state.Follower}
				server.electionTimer.Stop()
			}
		}
		if server.State.GetServerState() == state.Leader {
			leaderToFollower, errL2F := server.State.DiscoverServerWithHigherTerm()
			if errL2F == nil {
				server.State.SetServerState(leaderToFollower)
				server.State.WatchStateChange <- state.StateTransition{From: state.Leader, To: state.Follower}
			}
		}
	}

	// TODO: catch error for len(message.Terms) != len(message.Indexes) != len(message.Commands)
	// Ongaro & Ousterhout pg308.
	// AppendEntries RPC | Receiver implementation (4)
	var newEntryIndex types.LogIndex
	var newEntryTerm types.Term
	var newEntryCommand types.Command
	for i := 0; i < len(message.Terms); i++ {
		newEntryIndex = types.LogIndex(message.Indexes[i])
		newEntryTerm = types.Term(message.Terms[i])
		newEntryCommand = types.Command(message.Commands[i])
		err := server.State.AppendEntry(newEntryIndex, newEntryTerm, newEntryCommand)
		if err != nil {
			log.Println("AppendEntriesServerError:", err)
		}
	}

	reply := pb.AppendEntriesReply{
		Term:    int32(server.State.CurrentTerm),
		Success: true,
	}

	// Ongaro & Ousterhout pg308.
	// AppendEntries RPC | Receiver implementation (5)
	if message.LeaderCommit > int32(server.State.GetCommitIndex()) {
		lastNewEntryIndex := message.Indexes[len(message.Indexes)-1]
		if message.LeaderCommit < lastNewEntryIndex {
			server.State.SetCommitIndex(types.CommitIndex(message.LeaderCommit))
		} else {
			server.State.SetCommitIndex(types.CommitIndex(lastNewEntryIndex))
		}
	}
	return &reply, nil
}
