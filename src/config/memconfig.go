package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sleepy-hotstuff/src/logging"
)

var homepath string

var host string
var port string

type MemConfig struct {
	Host string `json:"host"` // IP address.
	Port string `json:"port"` // Port number
}

func SetHomeDir() {
	exepath, err := os.Executable()
	if err != nil {
		p := fmt.Sprintf("[Consortiumapi Error]  Failed to get path for the executable")
		logging.PrintLog(true, logging.ErrorLog, p)
		os.Exit(1)
	}

	p1 := path.Dir(exepath)
	homepath = path.Dir(p1)
}

/*
Load configuration for a new node joining the system (as a new replica)
*/
func LoadJoinConfig() {

	defaultFileName := homepath + "/etc/join.json"
	f, err := os.Open(defaultFileName)
	if err != nil {
		p := fmt.Sprintf("[Configuration Error] Failed to open config file: %v", err)
		logging.PrintLog(FetchVerbose(), logging.ErrorLog, p)
		os.Exit(1)
	}
	defer f.Close()
	var mConfig MemConfig
	byteValue, _ := ioutil.ReadAll(f)

	json.Unmarshal(byteValue, &mConfig)

	host = mConfig.Host
	port = ":" + mConfig.Port
}

/*
Return host ip address
*/
func GetHost() string {
	return host
}

/*
Return port number
*/
func GetPort() string {
	return port
}

/*
Return address of the node
*/
func GetAddress() string {
	return host + port
}

/*
Update the configuration file based on the updated configuration.
*/
func WriteToConfigFile(filename string, system System) {
	data, error := json.MarshalIndent(system, "", "   ")
	if error != nil {
		p := fmt.Sprintf("[Configuration Error] Failed to marshal json file to config: %v", error)
		logging.PrintLog(FetchVerbose(), logging.ErrorLog, p)
		os.Exit(1)
	}
	error = ioutil.WriteFile(filename, data, 0777)
	if error != nil {
		p := fmt.Sprintf("[Configuration Error] Unable to write to %s file.", filename)
		logging.PrintLog(FetchVerbose(), logging.ErrorLog, p)
		os.Exit(1)
	}
}

/*
Add a new replica to the configuration file.
*/
func AddReplica(id string, host string, port string) {
	defaultFileName := homepath + "/etc/conf.json"
	f, err := os.Open(defaultFileName)
	if err != nil {
		p := fmt.Sprintf("[Configuration Error] Failed to open config file: %v", err)
		logging.PrintLog(FetchVerbose(), logging.ErrorLog, p)
		os.Exit(1)
	}
	defer f.Close()
	var system System
	byteValue, _ := ioutil.ReadAll(f)

	json.Unmarshal(byteValue, &system)

	replica := &Replica{
		ID:   id,
		Host: host,
		Port: port[1:],
	}
	system.Replicas = append(system.Replicas, *replica)

	WriteToConfigFile(defaultFileName, system)
}

/*
Delete a replica from the configuration.
*/
func DeleteReplica(id string) {
	defaultFileName := homepath + "/etc/conf.json"
	f, err := os.Open(defaultFileName)
	if err != nil {
		p := fmt.Sprintf("[Configuration Error] Failed to open config file: %v", err)
		logging.PrintLog(FetchVerbose(), logging.ErrorLog, p)
		os.Exit(1)
	}
	defer f.Close()
	var system System
	byteValue, _ := ioutil.ReadAll(f)

	json.Unmarshal(byteValue, &system)

	var replicas []Replica
	for i := 0; i < len(system.Replicas); i++ {
		if system.Replicas[i].ID != id {
			replicas = append(replicas, system.Replicas[i])
		}
	}
	system.Replicas = replicas

	WriteToConfigFile(defaultFileName, system)

}
