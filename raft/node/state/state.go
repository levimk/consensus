package state

import (
	"errors"
	"fmt"
	"log"
	"raft/types"
)

type NodeState struct {
	ServerState
	persistentState
	volatileState
	volatileLeaderState
	WatchStateChange chan StateTransition
}

// ----- FACTORY -----
func NewState() NodeState {
	return NodeState{
		newServerState(),
		newPersistentState(),
		newVolatileState(),
		newVolatileLeaderState(),
		make(chan StateTransition),
	}
}

// ####----- FUNCTIONS -----####
func (state *NodeState) GetCurrentTerm() types.Term {
	return state.CurrentTerm
}

func (state *NodeState) GetLastLogIndex() types.LogIndex {
	return types.LogIndex(len(state.Log) - 1)
}

func (state *NodeState) GetPrevLogIndexForPeer(id types.NodeId) (types.LogIndex, error) {
	if nextLogIndex, pres := state.nextIndex[id]; pres {
		return nextLogIndex - 1, nil
	}
	err := errors.New(fmt.Sprintf("no nextLogIndex for [%d]", id))
	return ID_NOT_FOUND_LOG_INDEX, err
}

func (state *NodeState) GetPrevLogTermForPeer(id types.NodeId) (types.Term, error) {
	if nextLogIndex, pres := state.nextIndex[id]; pres {
		// TODO: update this to use getters from non-volatile state
		return state.Log[nextLogIndex-1].Term, nil
	}
	err := errors.New(fmt.Sprintf("(lastLogTerm) no lastLogIndex for [%d]", id))
	return ID_NOT_FOUND_TERM, err
}

func (state *NodeState) GetEntriesSince(index types.LogIndex) (types.Entries, error) {
	_index := int(index)
	numEntries := len(state.Log) - int(index) - 1
	entries := make(types.Entries, numEntries)
	for i := _index; i < _index+numEntries; i++ {
		// TODO: update this to use getters from non-volatile state
		entry, err := state.GetEntryAt(i)
		if err != nil {
			log.Panicln(err)
			return types.Entries{}, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (state *NodeState) GetMatchIndexForPeer(id types.NodeId) (types.LogIndex, error) {
	if matchIndex, pres := state.matchIndex[id]; pres {
		return matchIndex, nil
	}
	err := errors.New(fmt.Sprintf("no matchINdex for [%d]", id))
	return ID_NOT_FOUND_LOG_INDEX, err
}

func (state *NodeState) DeleteEntriesFromIndexOnwards(index types.LogIndex) {
	if index == 0 {
		state.Log = make(types.Entries, 0)
	} else {
		state.Log = state.Log[0:index]
	}
}

// ####----- LEADER INITIALISATION -----####
func (state *NodeState) ReinitialiseLeaderState() {
	if state.ServerState != Leader {
		log.Fatal("Unable to call ReinitialiseLeaderState() from non-leader")
	}
	state.volatileLeaderState = newVolatileLeaderState()
}

func (state *NodeState) MarkHasSentInitialHeartbeat() {
	if state.ServerState != Leader {
		log.Fatal("Unable to call MarkHasSentInitialHeartbeat() from non-leader")
	}
	state.volatileLeaderState.hasSentInitial = true
}

func (state *NodeState) HasSentInitialHeartbeat() bool {
	if state.ServerState != Leader {
		log.Fatal("Unable to call HasSentInitialHeartbeat() from non-leader")
	}
	return state.volatileLeaderState.hasSentInitial
}
