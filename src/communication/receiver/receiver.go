/*
Receiver functions.
It implements all the gRPC services defined in communication.proto file.
*/

package receiver

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sleepy-hotstuff/src/communication"
	"sleepy-hotstuff/src/config"
	"sleepy-hotstuff/src/consensus"
	"sleepy-hotstuff/src/cryptolib"
	"sleepy-hotstuff/src/db"
	"sleepy-hotstuff/src/hacss"
	logging "sleepy-hotstuff/src/logging"
	pb "sleepy-hotstuff/src/proto/communication"
	"sleepy-hotstuff/src/utils"
	"sync"

	"google.golang.org/grpc"
)

var id string
var wg sync.WaitGroup
var sleepTimerValue int
var con int

type server struct {
	pb.UnimplementedSendServer
}

type reserver struct {
	pb.UnimplementedSendServer
}

/*
Handle replica messages (consensus normal operations)
*/
func (s *server) SendMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	//go handler.HandleByteMsg(in.GetMsg())
	return &pb.Empty{}, nil
}

// SendRequest is only invoked by clients.
func (s *server) SendRequest(ctx context.Context, in *pb.Request) (*pb.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		// if the ctx is cancelled or timeout, this message has no need to process.
		return nil, err
	}
	return HandleRequest(in)
}

func (s *reserver) SendRequest(ctx context.Context, in *pb.Request) (*pb.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		// if the ctx is cancelled or timeout, this message has no need to process.
		return nil, err
	}
	return HandleRequest(in)
}

// Handle the request received in SendRequest.
// Only clients can send Request.
func HandleRequest(in *pb.Request) (*pb.RawMessage, error) {
	/*h := cryptolib.GenHash(in.GetRequest())
	rtype := in.GetType()*/

	/*go handler.HandleRequest(in.GetRequest(), utils.BytesToString(h))


	replies := make(chan []byte)
	go handler.GetResponseViaChan(utils.BytesToString(h), replies)
	reply := <-replies*/
	wtype := in.GetType()
	switch wtype {
	case pb.MessageType_WRITE_BATCH:
		consensus.HandleBatchRequest(in.GetRequest())
		reply := []byte("batch rep")

		return &pb.RawMessage{Msg: reply}, nil
	case pb.MessageType_RECONSTRUCT:
		go hacss.HandleReconstructMsg(in.GetRequest())

		reply := []byte("rep")
		return &pb.RawMessage{Msg: reply}, nil
	case pb.MessageType_TEST_HACSS:
		go consensus.HandleTestHacssMsg(in.GetRequest())
		reply := []byte("rep")
		return &pb.RawMessage{Msg: reply}, nil
	default:
		h := cryptolib.GenHash(in.GetRequest())
		// hash is actually not used in consensus.HandleRequest
		go consensus.HandleRequest(in.GetRequest(), utils.BytesToString(h))

		reply := []byte("rep")

		return &pb.RawMessage{Msg: reply}, nil
		// respond before the block is committed
	}
}

func (s *server) ABASendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	switch consensus.ConsensusType(con) {

	default:
		log.Fatalf("consensus type %v not supported in ABASendByteMsg function", con)
	}

	return &pb.Empty{}, nil
}

func (s *server) HACSSSendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	go hacss.HandleHACSSMsg(in.GetMsg())
	return &pb.Empty{}, nil
}

func (s *server) HotStuffSendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	if err := ctx.Err(); err != nil {
		// if the ctx is cancelled or timeout, this message has no need to process.
		return nil, err
	}

	//if consensus.SleepFlag.Get() == 0 {
	//	// In the test, a sleeping replica does nothing when it receives a message.
	//	return &pb.Empty{}, nil
	//}

	go consensus.HandleQCByteMsg(in.GetMsg())
	consensus.MsgQueue.AppendAndTrimToMaxSize(in.GetMsg())
	db.PersistValue("MsgQueue", &consensus.MsgQueue, db.PersistAll)
	return &pb.Empty{}, nil
}

/*
Handle join requests for both static membership (initialization) and dynamic membership.
Each replica gets a conformation for a membership request.
*/
func (s *server) Join(ctx context.Context, in *pb.RawMessage) (*pb.RawMessage, error) {

	reply := []byte("hi")
	result := true

	// go consensus.HandleMembershipRequest(in.GetMsg())
	return &pb.RawMessage{Msg: reply, Result: result}, nil
}

/*
Register rpc socket via port number and ip address
*/
func register(port string, splitPort bool) {
	lis, err := net.Listen("tcp", port)

	if err != nil {
		p := fmt.Sprintf("[Communication Receiver Error] failed to listen %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		os.Exit(1)
	}
	if config.FetchVerbose() {
		p := fmt.Sprintf("[Communication Receiver] listening to port %v", port)
		logging.PrintLog(config.FetchVerbose(), logging.NormalLog, p)
	}

	log.Printf("ready to listen to port %v", port)
	go serveGRPC(lis, splitPort)

}

/*
Have serve grpc as a function (could be used together with goroutine)
*/
func serveGRPC(lis net.Listener, splitPort bool) {
	defer wg.Done()

	if splitPort {

		s1 := grpc.NewServer(grpc.MaxRecvMsgSize(52428800), grpc.MaxSendMsgSize(52428800))

		pb.RegisterSendServer(s1, &reserver{})
		log.Printf("listening to split port")
		if err := s1.Serve(lis); err != nil {
			p := fmt.Sprintf("[Communication Receiver Error] failed to serve: %v", err)
			logging.PrintLog(true, logging.ErrorLog, p)
			os.Exit(1)
		}

		return
	}

	s := grpc.NewServer(grpc.MaxRecvMsgSize(52428800), grpc.MaxSendMsgSize(52428800))

	pb.RegisterSendServer(s, &server{})

	// In the method Serve, the listener will invoke the method Accept
	// to wait for a connection in a blocking way.
	if err := s.Serve(lis); err != nil {
		p := fmt.Sprintf("[Communication Receiver Error] failed to serve: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		os.Exit(1)
	}

}

/*
Start receiver parameters initialization
*/
func StartReceiver(rid string, cons bool, mem string) {
	id = rid
	logging.SetID(rid)

	config.LoadConfig()
	logging.SetLogOpt(config.FetchLogOpt())
	con = config.Consensus()

	sleepTimerValue = config.FetchSleepTimer()
	if cons {
		consensus.StartHandler(rid, mem)
	}

	if config.SplitPorts() {
		//wg.Add(1)
		go register(communication.GetPortNumber(config.FetchPort(rid)), true)
	}
	wg.Add(1)
	register(config.FetchPort(rid), false)
	wg.Wait()

}
