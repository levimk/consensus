package consensus

import "raft/types"

type IConsensus interface {
	Submit(command types.Command) error
	// Run(transport Transport, output chan<- CommitEvent)
}
