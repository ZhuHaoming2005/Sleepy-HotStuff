/*
Sender functions.
It implements all sending functions for replicas.
*/

package sender

import (
	"context"
	"fmt"
	"sleepy-hotstuff/src/communication"
	"sleepy-hotstuff/src/config"
	logging "sleepy-hotstuff/src/logging"
	"sleepy-hotstuff/src/message"
	pb "sleepy-hotstuff/src/proto/communication"
	"sleepy-hotstuff/src/utils"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
)

var id int64
var idstring string
var err error

// var completed map[string]bool
var verbose bool

var wg sync.WaitGroup

var broadcastTimer int
var sleepTimerValue int
var reply []byte

var dialOpt []grpc.DialOption
var connections communication.AddrConnMap

func BuildConnection(ctx context.Context, nid string, address string) bool {
	p := fmt.Sprintf("building a connection with %v", nid)
	logging.PrintLog(verbose, logging.NormalLog, p)

	/*if config.CommOption() == "TLS" {
		dialOpt = communication.GetDialOption()
	}*/
	conn, err := grpc.DialContext(ctx, address, dialOpt...)

	if err != nil {
		p := fmt.Sprintf("[Communication Sender Error] failed to bulid a connection with %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return false
	}
	c := pb.NewSendClient(conn)

	connections.Insert(address, c)
	connections.InsertID(address, nid)
	return true
}

func ByteSend(msg []byte, address string, msgType message.TypeOfMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(broadcastTimer)*time.Millisecond)
	defer cancel()

	if address == "" {
		return
	}
	nid := config.FetchReplicaID(address)
	c, built := connections.Get(address)
	existnid := connections.GetID(address)

	if !built || c == nil || nid != existnid {
		suc := BuildConnection(ctx, nid, address)
		if !suc {
			p := fmt.Sprintf("[Communication Sender Error] did not connect to node %s, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)

			communication.NotLive(nid)
			broadcastTimer = broadcastTimer * 2

			return
		} else {
			c, _ = connections.Get(address)
		}
	}

	switch msgType {
	case message.ABA_ALL:
		_, err = c.ABASendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.HACSS_ALL:
		_, err = c.HACSSSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			return
		}
	case message.HotStuff_Msg:
		_, err = c.HotStuffSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			return
		}
	case message.Rondo_Msg:
		_, err = c.Join(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			return
		}
	default:
		log.Fatalf("message type %v not supported", msgType)
	}
}

func RBCByteBroadcast(msg []byte) {

	request, err := message.SerializeWithSignature(id, msg)
	if err != nil {
		logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to sign the message")
		return
	}
	nodes := FetchNodesFromConfig()

	for i := 0; i < len(nodes); i++ {
		nid := nodes[i]
		if nid == idstring {
			continue
		}
		if communication.IsNotLive(nid) {
			p := fmt.Sprintf("[Communication Sender] Replica %v is not live, don't send message to it", nid)
			logging.PrintLog(verbose, logging.NormalLog, p)
			continue
		}
		go ByteSend(request, config.FetchAddress(nid), message.HotStuff_Msg)
	}
}

func JoinBroadcast(msg []byte) {
	request, err := message.SerializeWithSignature(id, msg)
	if err != nil {
		logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to sign the message")
		return
	}
	nodes := FetchNodesFromConfig()

	for i := 0; i < len(nodes); i++ {
		nid := nodes[i]
		if nid == idstring {
			continue
		}
		if communication.IsNotLive(nid) {
			p := fmt.Sprintf("[Communication Sender] Replica %v is not live, don't send message to it", nid)
			logging.PrintLog(verbose, logging.NormalLog, p)
			continue
		}
		go ByteSend(request, config.FetchAddress(nid), message.Rondo_Msg)
	}
}

func MACBroadcast(msg []byte, mtype message.ProtocolType) {

	nodes := FetchNodesFromConfig()

	for i := 0; i < len(nodes); i++ {
		nid := nodes[i]

		dest, _ := utils.StringToInt64(nid)
		request, err := message.SerializeWithMAC(id, dest, msg)
		if err != nil {
			logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to generate MAC")
			continue
		}

		if communication.IsNotLive(nid) {
			p := fmt.Sprintf("[Communication Sender] Replica %v is not live, don't send message to it", nid)
			logging.PrintLog(verbose, logging.NormalLog, p)
			continue
		}
		switch mtype {
		case message.ABA:
			go ByteSend(request, config.FetchAddress(nid), message.ABA_ALL)
		case message.HACSS:
			go ByteSend(request, config.FetchAddress(nid), message.HACSS_ALL)
		}

	}
}

func SendToNode(msg []byte, dest int64, mtype message.ProtocolType) {

	nid := utils.Int64ToString(dest)

	if id == dest {
		return 
	}

	
	switch mtype {
	case message.HACSS:
		request, err := message.SerializeWithMAC(id, dest, msg)
		if err != nil {
			logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to generate MAC")
			return
		}

		go ByteSend(request, config.FetchAddress(nid), message.HACSS_ALL)
	case message.HotStuff:
		request, err := message.SerializeWithSignature(id, msg)
		if err != nil {
			logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to sign the message")
			return
		}
		go ByteSend(request, config.FetchAddress(nid), message.HotStuff_Msg)

	default:
		log.Printf("Not supported type: %v", mtype)
	}

}

/*
Used for membership protocol to fetch list of nodes
Output

	[]string: a list of nodes (in the string type)
*/
func FetchNodesFromConfig() []string {
	return config.FetchNodes()
}

func StartSender(rid string) {
	log.Printf("Starting sender %v", rid)
	config.LoadConfig()
	verbose = config.FetchVerbose()
	idstring = rid

	id, err = utils.StringToInt64(rid) // string to int64
	if err != nil {
		p := fmt.Sprintf("[Communication Sender Error] Replica id %v is not valid. Double check the configuration file", id)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	// Set up a connection to the server.

	dialOpt = []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
		//grpc.WithKeepaliveParams(kacp),
	}

	connections.Init()

	verbose = config.FetchVerbose()
	communication.StartConnectionManager()
	broadcastTimer = config.FetchBroadcastTimer()
	sleepTimerValue = config.FetchSleepTimer()
}

func SetId(newnid int64) {
	id = newnid
}
