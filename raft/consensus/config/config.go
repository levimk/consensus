package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"raft/types"

	"gopkg.in/yaml.v3"
)

type RaftConfig struct {
	Id     types.NodeId
	Server struct {
		Hostname      string `yaml:"hostname"`
		Port          int    `yaml:"port"`
		Communication struct {
			Protocol string `yaml:"protocol"`
		} `yaml:"communication"`
		Keys struct {
			PublicKey  string `yaml:"public"`
			PrivateKey string `yaml:"private"`
		} `yaml:"keys"`
		Peers []string `yaml:"peers"`
	} `yaml:"server"`
}

func NewConfig(id types.NodeId, configPath string) (RaftConfig, error) {
	log.Println(fmt.Sprintf("[%d] configuring using %s/node_%d.yaml", id, configPath, id))
	filename := fmt.Sprintf("%s/node_%d.yaml", configPath, id)
	configFile, err := ioutil.ReadFile(filename)

	if err != nil {
		fmt.Println("Raft config file error")
		return RaftConfig{}, err
	}
	config := RaftConfig{
		Id: id,
	}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		fmt.Println("Raft config yaml error")
		return RaftConfig{}, err
	}
	log.Println(fmt.Sprintf("[%d] configuration complete:", id))
	fmt.Println(config)
	return config, nil
}

func (config RaftConfig) String() string {
	serverStr := fmt.Sprintf("\n\t(%s)%s:%d",
		config.Server.Communication.Protocol,
		config.Server.Hostname,
		config.Server.Port)
	keysStr := fmt.Sprintf("\n\t%s\n\t%s", config.Server.Keys.PrivateKey, config.Server.Keys.PublicKey)
	peersStr := fmt.Sprintf("\n\t%v", config.Server.Peers)
	return fmt.Sprintf("Raft[%d] config:%s%s%s", config.Id, serverStr, keysStr, peersStr)
}
