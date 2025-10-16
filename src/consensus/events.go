package consensus

import (
	"fmt"
	"log"
	"sleepy-hotstuff/src/config"
	"sleepy-hotstuff/src/db"
	"sleepy-hotstuff/src/logging"
	"sleepy-hotstuff/src/quorum"
	"sleepy-hotstuff/src/utils"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

var verbose bool //verbose level
var id int64     //id of server
var iid int      //id in type int, start a RBC using it to instanceid
var errs error
var queue Queue         // cached client requests
var queueHead QueueHead // hash of the request that is in the first place of the queue
var sleepTimerValue int // sleeptimer for the while loop that continues to monitor the queue or the request status
var consensus ConsensusType
var rbcType RbcType
var n int
var members []int
var t1 int64
var baseinstance int

var batchSize int
var requestSize int

var midTime map[int]int64

var MsgQueue Queue // record the consensus messages received so far.

func ExitEpoch() {
	t2 := utils.MakeTimestamp()
	if (t2 - t1) == 0 {
		log.Printf("Latancy is zero!")
		return
	}
	if outputSize.Get() == 0 {
		log.Printf("Finish zero instacne!")
		return
	}
	log.Printf("*****epoch ends, latency %v ms.", t2-t1)
	//log.Printf("*****epoch ends with  output size %v, latency %v ms, throughput %d, quorum tps %d", outputSize.Get(), t2-t1, int64(outputSize.Get()*batchSize*1000)/(t2-t1), int64(quorum.QuorumSize()*batchSize*1000)/(t2-t1))
	t3, exi := midTime[epoch.Get()]
	if !exi {
		log.Printf("can not get mid time for epoch %v", epoch.Get())
	} else {
		p := fmt.Sprintf("%v %v %v %v %v %v %v %v %v", quorum.FSize(), batchSize, int64(outputSize.Get()*batchSize*1000)/(t2-t1), int64(quorum.QuorumSize()*batchSize*1000)/(t2-t1), t2-t1, requestSize, outputSize.Get(), t3-t1, t2-t3)
		logging.PrintLog(true, logging.EvaluationLog, p)
	}

}

func CaptureRBCLat() {
	t3 := utils.MakeTimestamp()
	if (t3 - t1) == 0 {
		log.Printf("Latancy is zero!")
		return
	}
	log.Printf("*****RBC phase ends with %v ms", t3-t1)

}

func CaptureLastRBCLat() {
	t3 := utils.MakeTimestamp()
	if (t3 - t1) == 0 {
		log.Printf("Latancy is zero!")
		return
	}
	log.Printf("*****Final RBC phase ends with %v ms", t3-t1)

}

// This func is used to grab cached requests from clients and propose new proposals.
// So if the node is not a leader, it will leave this func.
// Modify: when invoking this func, users need to input the view number associated with this monitor.
// This monitor will return when its view number is not the current view.
func RequestMonitor(v int) {
	if consensus == HotStuff {
		// comments: Now we only test the view change of hotstuff.
		for queue.IsEmpty() && LocalView() == 0 {
			// wait until the first client sends its first request.
			// then we start the rotatingTimer.
			// In other words, we view the time the first request is received
			//as the beginning of the system.
			// log.Printf("wait for new requests in queue.")
			continue
		}
		if config.IsViewChangeMode() {
			StartRotatingTimer(v)
		}
	}

	for {
		if v != LocalView() {
			return
		}
		switch consensus {
		case HotStuff:
			sleepLock.RLock()
			if curStatus.Get() == SLEEPING {
				sleepLock.RUnlock()
				return
			}
			if LeaderID(v) != int(id) {
				sleepLock.RUnlock()
				return
			}

			if !(curStatus.Get() == READY && (awaitingDecisionCopy.GetLen() > 0 || !queue.IsEmpty())) {
				// awaitingDecisionCopy.GenLen seems to be always > 0,
				// except the initial period of a leader.

				if awaitingDecisionCopy.GetLen() == 0 {
					log.Printf("[Error] awaitingDecisionCopy.GetLen() == 0")
				}
				if curStatus.Get() != READY {
					//log.Printf("[Error] curStatus is %v, is not READY!", curStatus.Get())
				}

				// Here, wait for sleepTimerValue is not a suitable method to wait until curState == READY
				// It will add the block latency.
				if queue.IsEmpty() {
					// log.Printf("sleep %d ms", 5*sleepTimerValue)
					time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
				} else {
					time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
				}
				sleepLock.RUnlock()
				continue
			}

			curStatus.Set(PROCESSING)
			batch := queue.GrabWithMaxLenAndClear()
			db.PersistValue("queue", &queue, db.PersistAll)
			log.Println("batchSize:", len(batch))
			StartHotStuff(batch)
			sleepLock.RUnlock()
		}
	}
}

func HandleRequest(request []byte, hash string) {
	//log.Printf("Handling request")
	//rawMessage := message.DeserializeMessageWithSignature(request)
	//m := message.DeserializeClientRequest(rawMessage.Msg)

	/*if !cryptolib.VerifySig(m.ID, rawMessage.Msg, rawMessage.Sig) {
		log.Printf("[Authentication Error] The signature of client request has not been verified.")
		return
	}*/
	//log.Printf("Receive len %v op %v\n",len(request),m.OP)
	batchSize = 1
	requestSize = len(request)
	queue.Append(request)
	db.PersistValue("queue", &queue, db.PersistAll)
}

func HandleBatchRequest(requests []byte) {
	requestArr := DeserializeRequests(requests)
	//var hashes []string
	Len := len(requestArr)
	log.Printf("Handling batch requests with len %v\n", Len)
	//for i:=0;i<Len;i++{
	//	hashes = append(hashes,string(cryptolib.GenHash(requestArr[i])))
	//}
	//for i:=0;i<Len;i++{
	//	HandleRequest(requestArr[i],hashes[i])
	//}
	/*for i:=0;i<Len;i++{
		rawMessage := message.DeserializeMessageWithSignature(requestArr[i])
		m := message.DeserializeClientRequest(rawMessage.Msg)

		if !cryptolib.VerifySig(m.ID, rawMessage.Msg, rawMessage.Sig) {
			log.Printf("[Authentication Error] The signature of client logout request has not been verified.")
			return
		}
	}*/
	batchSize = Len
	requestSize = len(requestArr[0])
	queue.AppendBatch(requestArr)
	db.PersistValue("queue", &queue, db.PersistAll)
}

func DeserializeRequests(input []byte) [][]byte {
	var requestArr [][]byte
	msgpack.Unmarshal(input, &requestArr)
	return requestArr
}
