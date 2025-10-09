package consensus

import (
	"sleepy-hotstuff/src/config"
	"sleepy-hotstuff/src/utils"
	"sleepy-hotstuff/src/logging"
	"sleepy-hotstuff/src/message"
	pb "sleepy-hotstuff/src/proto/communication"
	"sleepy-hotstuff/src/cryptolib"
	"sleepy-hotstuff/src/communication/sender"
	"log"
	"os"
    "fmt"
)

var tmembers []int
var identifier []byte         //unique identifier for identify a node

var memqueue Queue         // cached membership request
var memqueueHead QueueHead // hash of the request that is in the fist place of the memqueue
var tmHost utils.IntStringMap //ip addresses of the nodes
var tmPort utils.IntStringMap // port numbers of the nodes

type MemOption int


const (
	Mem MemOption = 0
	TM  MemOption = 1
)

func InitializeMembership(){
	nodes := config.FetchNodes()
	for i := 0; i < len(nodes); i++ {
		nid, _ := utils.StringToInt(nodes[i])
		members = append(members, nid)
	}

}


func MemBershipRequest(mem string){
	switch mem{
	case "0":
		return 
	case "1":
		log.Printf("[Membership] joining the system")
		JoinSystem()
	case "2":
		log.Printf("[Membership] leaving the system")
	default:
		log.Printf("[Membership] membership request type not supported")
		return 
	}
}


// Provide information to join the system
func PrepareMembershipInfo() message.MembershipInfo {
	config.LoadJoinConfig()

	pk := cryptolib.LoadNewPubKey()
	m := message.MembershipInfo{
		IP:         config.GetHost(),
		Port:       config.GetPort(),
		TS:         utils.MakeTimestamp(),
		Key:        pk,
	}

	identifier = m.GetIdentifier()
	return m
}



func JoinSystem() {
	m := message.JoinMessage{
		Mtype:         pb.MessageType_JOIN,
		Source:        id,
		MInfo:         PrepareMembershipInfo(), 
	}

	//go MonitorCatchUpMessage()

	msgbyte, err := m.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Join Error] Not able to serialize Join message %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		os.Exit(1)
	}

	
	sender.JoinBroadcast(msgbyte)
}

// Update membership information for temporary members
func UpdateTemMembershipInfo(memops []pb.RawMessage) {
	if len(memops) == 0 {
		return
	}

	for i := 0; i < len(memops); i++ {
		TMManagement(memops[i].GetMsg())
	}

}

//Add new node temporarily to TM
func TMManagement(inputMsg []byte) {
	memMsg := message.DeserializeMemMessage(inputMsg)
	InsertNode(memMsg.ID, TM)
	
	tmHost.Insert(memMsg.ID, memMsg.JMsg.MInfo.IP)
	tmPort.Insert(memMsg.ID, memMsg.JMsg.MInfo.Port)
	return
}


//insert a node
func InsertNode(id int, mo MemOption) {
	switch mo {
	case TM:
		tmembers = append(tmembers,id)
	case Mem:
		members = append(members, id)
		UpdateNumOfNodes(n + 1)
	}
}


//update number of nodes
func UpdateNumOfNodes(numOfNodes int) {
	n = numOfNodes
}
