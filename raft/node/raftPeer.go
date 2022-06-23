package node

import (
	"fmt"
	"log"
	pb "raft/infrastructure"
	rt "raft/types"

	"google.golang.org/grpc"
)

type RaftPeer struct {
	Id         rt.NodeId
	Port       int
	Hostname   string
	Client     pb.RaftClient
	connection *grpc.ClientConn
}

func (peer *RaftPeer) GetAddress() string {
	return fmt.Sprintf("%s:%d", peer.Hostname, peer.Port)
}

func (node *RaftNode) AddPeer(id rt.NodeId, hostname string, port int) {
	node.mtx.Lock()
	defer node.mtx.Unlock()
	if _, found := node.peers[id]; found {
		log.Fatalf("[%d] already has peer [%d]", node.Id, id)
		return
	}

	var options []grpc.DialOption
	address := fmt.Sprintf("%s:%d", hostname, port)
	conn, err := grpc.Dial(address, options...)
	if err != nil {
		log.Fatalf("Failed to create connection: %v", err)
	}

	peer := RaftPeer{
		Id:         id,
		Port:       port,
		Hostname:   hostname,
		Client:     pb.NewRaftClient(conn),
		connection: conn,
	}
	node.peers[id] = &peer
}

func (node *RaftNode) RemovePeer(id rt.NodeId) {
	node.mtx.Lock()
	defer node.mtx.Unlock()
	if peer, found := node.peers[id]; found {
		peer.connection.Close()
		delete(node.peers, id)
	}
	log.Fatalf("[%d] has no peer [%d]", node.Id, id)
	return
}
