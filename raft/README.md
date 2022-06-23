Consensus:

- Multiple servers agree on a value
- Once a decision is made it is final
  Fault-tolerance
- Majority: only need (N/2) + 1 servers online to progress
  Replicated state machine
- Each server has a state machine (SM)
- Each server has a log
- If each server has the same log then they also have the same SM
  The State Machine _is_ the component we want to make Fault-Tolerant
  To the client, it looks like it is interacting with a single reliable
  state machine, not a replicated cluster.
  Consensus ensures that if command Ci is the ith command then no server
  will apply a command other than Ci at log position i
