package consensus

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"sleepy-hotstuff/src/communication/sender"
	"sleepy-hotstuff/src/config"
	"sleepy-hotstuff/src/cryptolib"
	"sleepy-hotstuff/src/db"
	"sleepy-hotstuff/src/logging"
	"sleepy-hotstuff/src/message"
	pb "sleepy-hotstuff/src/proto/communication"
	"sleepy-hotstuff/src/quorum"
	"sleepy-hotstuff/src/utils"
	"sync"
	"time"
)

var s int
var f int
var gat bool
var reqHash utils.ByteValue // record the latest recover request.
var hView utils.IntValue
var sleepLock sync.RWMutex
var recLock sync.Mutex
var recBuffer utils.StringIntMap

func SleepyHotstuffConfig() (string, error) {
	var err error = nil
	conf := ""
	f = config.FetchNumOfMal()
	s = config.FetchNumOfSleepy()
	if 3*f >= n {
		ss := fmt.Sprintf("%d replicas can not tolerate %d faults", n, f)
		err = errors.New(ss)
		return conf, err
	}

	gat = config.GAT()
	plevel := db.PersistLevelType(config.PersistLevel())
	if plevel == db.NoPersist {
		if gat {
			conf = "3f+s+1"
		} else {
			conf = "3f+2s+1"
		}
	} else {
		if gat {
			conf = "3f+1"
		} else {
			conf = "3f+2s+1"
		}
	}
	switch conf {
	case "3f+1":
		if n != 3*f+1 {
			err = errors.New("[CONFIG Error]: n!=3f+s+1, double check your conf.json file!")
		}
	case "3f+s+1":
		if n != 3*f+s+1 {
			err = errors.New("[CONFIG Error]: n!=3f+s+1, double check your conf.json file!")
		}
	case "3f+2s+1":
		if n != 3*f+2*s+1 {
			err = errors.New("[CONFIG Error]: n!=3f+2s+1, double check your conf.json file!")
		}
	}
	return conf, err
}

func RecoveryProcess(recMode config.RecModeType) error {
	log.Printf("Start the recovery process.")
	if curStatus.Get() != SLEEPING {
		err := errors.New("[Recovery Error] The status before recovery is not SLEEPING!")
		return err
	}
	recLock.Lock()
	defer recLock.Unlock()
	reqHash.Init()
	hView.Set(-2) // init hView
	curStatus.Set(RECOVERING)
	switch recMode {
	case config.RecFromDisk:
		recoverFromDisk()
		return nil
	case config.NoRec:
		curStatus.Set(READY)
		// queue.Append(utils.StringToBytes("empty tx"))
		go RequestMonitor(0)
		log.Printf("recover to READY")
		return nil
	case config.RecKoala2:
		msg := message.HotStuffMessage{
			Mtype:  pb.MessageType_REC1,
			Source: id,
			TS:     utils.MakeTimestamp(),
		}
		msgbyte, err := msg.Serialize()
		if err != nil {
			log.Fatal(err)
		}

		//request, _ := message.SerializeWithSignature(id, msgbyte)
		reqHash.Set(cryptolib.GenHash(msgbyte))
		sender.RBCByteBroadcast(msgbyte)
		return nil
	default:
		log.Fatal("[Recovery Error] Unknown RecModeType!")
		return errors.New("[Recovery Error] Unknown RecModeType!")
	}
}

func HandleRec1Msg(content message.HotStuffMessage) {
	log.Printf("receive a REC1 msg from replica %v", content.Source)
	viewMux.Lock()
	contentByte, _ := content.Serialize()
	msg := message.HotStuffMessage{
		Mtype:  pb.MessageType_ECHO1,
		Source: id,
		View:   LocalView() - 1,
		Hash:   cryptolib.GenHash(contentByte),
		V:      timeoutBuffer.GetV(LocalView() - 1),
	}
	viewMux.Unlock()

	msgbyte, err := msg.Serialize()
	if err != nil {
		logging.PrintLog(true, logging.ErrorLog, "[ECHO1Message Error] Not able to serialize the message")
		return
	}
	go sender.SendToNode(msgbyte, content.Source, message.HotStuff)
}

func HandleEcho1Msg(content message.HotStuffMessage) {
	log.Printf("receive a ECHO1 msg from replica %v", content.Source)
	recLock.Lock()
	defer recLock.Unlock()
	if curStatus.Get() != RECOVERING {
		return
	}
	if !bytes.Equal(content.Hash, reqHash.Get()) {
		log.Printf("[ECHO1 Warning] The ECHO1 msg from replica %v does not match the latest reqHash.", content.Source)
		return
	}

	bufferLock.Lock()
	hashStr := utils.BytesToString(content.Hash)
	out, _ := GetBufferContent("ECHO1"+hashStr, BUFFER)
	if out == PREPARED {
		bufferLock.Unlock()
		return
	}

	if !VerifyTQC(content.View, content.V) {
		log.Printf("TQC in the ECHO1 msg from replica %v is not verified.", content.Source)
		bufferLock.Unlock()
		return
	}
	if content.View > hView.Get() {
		hView.Set(content.View)
	}

	num, exist := recBuffer.Get(hashStr)
	if !exist {
		num = 0
	}
	recBuffer.Insert(hashStr, num+1)
	if num+1 >= quorum.RecQuorumSize() {
		UpdateBufferContent("ECHO1"+hashStr, PREPARED, BUFFER)
		bufferLock.Unlock()
		// TQC received in ECHO1 msgs are not stored,
		//since a recovering replica will receive a TQC for a higher view before becoming READY.
		for LocalView() <= hView.Get()+2 {
			// a little issue: when this `for` is running, recLock keeping locked,
			// so the other receovery messages cannot be process.
			// But this seems to be safe,
			//as long as the local view inceasing process cannot be blocked by the recLock.
		}

		msg := message.HotStuffMessage{
			Mtype:  pb.MessageType_REC2,
			Source: id,
			TS:     utils.MakeTimestamp(),
			View:   LocalView(),
		}
		msgbyte, err := msg.Serialize()
		if err != nil {
			log.Fatal(err)
		}
		reqHash.Set(cryptolib.GenHash(msgbyte))
		sender.RBCByteBroadcast(msgbyte)
	} else {
		bufferLock.Unlock()
	}
}

func HandleRec2Msg(content message.HotStuffMessage) {
	log.Printf("receive a REC2 msg from replica %v", content.Source)
	for LocalView() < content.View {
	}

	cblock.Lock()
	qcbyte, _ := curBlock.Serialize()
	cblock.Unlock()
	lqcbyte, _ := lockedBlock.Serialize()
	contentByte, _ := content.Serialize()
	comBlockSer, _ := committedBlocks.Serialize()
	msg := message.HotStuffMessage{
		Mtype:     pb.MessageType_ECHO2,
		Source:    id,
		View:      LocalView(),
		Hash:      cryptolib.GenHash(contentByte),
		QC:        qcbyte,
		LQC:       lqcbyte,
		ComBlocks: comBlockSer,
	}

	msgbyte, err := msg.Serialize()
	if err != nil {
		logging.PrintLog(true, logging.ErrorLog, "[ECHO2Message Error] Not able to serialize the message")
		return
	}
	// time.Sleep(10 * time.Millisecond)
	go sender.SendToNode(msgbyte, content.Source, message.HotStuff)
}

func HandleEcho2Msg(content message.HotStuffMessage) {
	log.Printf("receive a ECHO2 msg from replica %v", content.Source)

	// TODO: the code below might be executed by multiple goroutines currently.
	// Thus, it is possible that a block is inserted when the block has existed in committedBlocks
	var comBlock utils.IntByteMap
	comBlock.Deserialize(content.ComBlocks)
	m := comBlock.GetAll()
	log.Printf("[Recovery] Update committedBlocks to that of replica %d", content.Source)
	for key := range m {
		if _, exist := committedBlocks.Get(key); !exist {
			committedBlocks.Insert(key, m[key])
		}
	}

	recLock.Lock()
	defer recLock.Unlock()
	if curStatus.Get() != RECOVERING {
		return
	}

	if !bytes.Equal(content.Hash, reqHash.Get()) {
		log.Printf("[ECHO2 Warning] The ECHO2 msg from replica %v does not match the latest reqHash.", content.Source)
		return
	}

	bufferLock.Lock()
	defer bufferLock.Unlock()
	hashStr := utils.BytesToString(content.Hash)
	out, _ := GetBufferContent("ECHO2"+hashStr, BUFFER)
	if out == PREPARED {
		return
	}

	qc := message.DeserializeQCBlock(content.QC)
	lqc := message.DeserializeQCBlock(content.LQC)

	if !VerifyQC(qc) || !VerifyQC(lqc) {
		log.Printf("perpareQC or LockQC in ECHO2 msg from replica %v is not verified.", content.Source)
		return
	}

	cblock.Lock()
	if qc.Hash != nil && qc.Height > curBlock.Height {
		// log.Printf("cb(%v) from %v is no lower than curBlock(%v)", cb.Height, content.Source, curBlock.Height)
		curBlock = qc
		UpdateSeq(qc.Height)
	}
	cblock.Unlock()
	lqcLock.Lock()
	if lqc.Hash != nil && lqc.Height > lockedBlock.Height {
		lockedBlock = lqc
	}
	lqcLock.Unlock()
	//if comBlock.GetLen() > committedBlocks.GetLen() {
	//	log.Printf("[Recovery] Update committedBlocks to that of replica %d", content.Source)
	//	committedBlocks.InsertAll(comBlock.GetAll())
	//	//
	//}
	//update the committedBlock

	num, exist := recBuffer.Get(hashStr)
	if !exist {
		num = 0
	}
	recBuffer.Insert(hashStr, num+1)
	if num+1 >= quorum.RecQuorumSize() {
		// wait for 100ms to collect more echo2 messages and update committedBlocks as much as possible
		time.Sleep(100 * time.Millisecond)
		UpdateBufferContent("ECHO2"+hashStr, PREPARED, BUFFER)
		curStatus.Set(READY)
		log.Printf("recover to READY")
	}
}

func recoverFromDisk() {
	plevel := db.PersistLevelType(config.PersistLevel())
	if plevel == db.PersistCritical {
		viewInt := utils.IntValue{}
		err := db.RecoverValue("view", &viewInt)
		if err != nil {
			log.Fatal(err)
		}
		err = db.RecoverValue("lockedBlock", &lockedBlock)
		if err != nil {
			log.Fatal(err)
		}
		curBlock = lockedBlock
		err = db.RecoverValue("committedBlocks", &committedBlocks)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("recover to the view %d", viewInt.Get()+1)
		// will be set to READY after the view change.
		// this implementation is not graceful, the viewMux should haven't been exposed to external modules, files or functions.
		viewMux.Lock()
		StartViewChange(viewInt.Get())
		viewMux.Unlock()
	} else if plevel == db.PersistAll {
		// TODO
		curStatus.Set(READY)
		log.Printf("recover to READY")
	} else {
		// if NoPersist: do nothing
		// else: not planned
	}
}

func viewChangeInRecovery(v int) {
	if v < LocalView() {
		// v is at most equal to LocalView(), can not be larger.
		return
	}
	SetView(v + 1)
	log.Printf("Starting view change to view %v", v+1)
}
