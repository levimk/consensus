package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"raft/consensus"
	"raft/types"

	"gopkg.in/yaml.v3"
)

type ClientServer struct {
	consensus consensus.IConsensus
	Config    ClientServerConfig
}

type ClientServerConfig struct {
	Server struct {
		Hostname string `yaml:"hostname"`
		Port     int32  `yaml:"port"`
	} `yaml:"server"`
}

func NewClientServer(consensus consensus.IConsensus) ClientServer {
	clientServer := ClientServer{
		consensus: consensus,
	}
	err := clientServer.config()
	if err != nil {
		log.Fatalln(err)
	}
	return clientServer
}

func (cs *ClientServer) config() error {
	config, err := ioutil.ReadFile("clientserver.yaml")
	if err != nil {
		fmt.Println("file must be `clientserver.yaml` in same directory as server")
		return err
	}
	err = yaml.Unmarshal(config, &cs.Config)
	if err != nil {
		return err
	}
	return err
}

func (cs *ClientServer) Run() {
	http.HandleFunc("/", cs.handleHome)
	http.HandleFunc("/add", cs.handleAdd)
	http.HandleFunc("/remove", cs.handleRemove)

	fmt.Println(fmt.Sprintf("ClientServer listening on %s:%d", cs.Config.Server.Hostname, cs.Config.Server.Port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cs.Config.Server.Port), nil))
}

func (cs *ClientServer) handleHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, Raft")
}

func (cs *ClientServer) handleAdd(w http.ResponseWriter, r *http.Request) {
	val := r.URL.Query().Get("val")
	clientId := r.URL.Query().Get("id")

	command := types.Command{
		ClientId: clientId,
		Command:  "add",
		Value:    val,
	}
	cs.consensus.Submit(command)
	fmt.Fprintf(w, "Client %s added %v", clientId, val)
}

func (cs *ClientServer) handleRemove(w http.ResponseWriter, r *http.Request) {
	val := r.URL.Query().Get("val")
	clientId := r.URL.Query().Get("id")

	command := types.Command{
		ClientId: clientId,
		Command:  "remove",
		Value:    val,
	}
	cs.consensus.Submit(command)
	fmt.Fprintf(w, "Client %s removed %v", clientId, val)
}
