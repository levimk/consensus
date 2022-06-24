package state

import (
	"log"
	"raft/types"
)

// TODO: re-initialize after elections

const ID_NOT_FOUND_LOG_INDEX types.LogIndex = -1
const ID_NOT_FOUND_TERM types.Term = -1

type volatileLeaderState struct {
	nextIndex      map[types.NodeId]types.LogIndex //  next log index to send to each server
	matchIndex     map[types.NodeId]types.LogIndex // highest log index known to be replicated on each server
	hasSentInitial bool
}

func newVolatileLeaderState() volatileLeaderState {
	return volatileLeaderState{
		nextIndex:      make(map[types.NodeId]types.LogIndex),
		matchIndex:     make(map[types.NodeId]types.LogIndex),
		hasSentInitial: false,
	}
}

func (vls *volatileLeaderState) DecrementNextIndex(nodeId types.NodeId) {
	if nextIndex, pres := vls.nextIndex[nodeId]; pres {
		if nextIndex == 0 {
			log.Fatalln("VolatileLeaderStateError: DecrementNextIndex - nextIndex is already 0 for node ", nodeId)
		}
		vls.nextIndex[nodeId] = nextIndex - 1
	} else {
		log.Fatalln("VolatileLeaderStateError: DecrementNextIndex - node id not found:", nodeId)
	}
}

func (vls *volatileLeaderState) UpdateNextIndex(nodeId types.NodeId, index types.LogIndex) {
	if _, pres := vls.nextIndex[nodeId]; pres {
		vls.nextIndex[nodeId] = index
	} else {
		log.Fatalln("VolatileLeaderStateError: UpdateNextIndex - node id not found:", nodeId)
	}
}

func (vls *volatileLeaderState) UpdateMatchIndex(nodeId types.NodeId, index types.LogIndex) {
	if _, pres := vls.matchIndex[nodeId]; pres {
		vls.matchIndex[nodeId] = index
	} else {
		log.Fatalln("VolatileLeaderStateError: UpdateMatchIndex - node id not found:", nodeId)
	}
}

func makeVolatileLeaderState(leaderLastLogIndex types.LogIndex, cluster []types.NodeId) volatileLeaderState {
	nextIndex := make(map[types.NodeId]types.LogIndex)
	matchIndex := make(map[types.NodeId]types.LogIndex)
	defaultNextIndex := leaderLastLogIndex + 1
	for _, nodeId := range cluster {
		nextIndex[nodeId] = defaultNextIndex
		matchIndex[nodeId] = 0
	}

	return volatileLeaderState{
		nextIndex:  nextIndex,
		matchIndex: matchIndex,
	}
}
