package consensus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sleepy-hotstuff/src/communication"
	"sleepy-hotstuff/src/communication/sender"
	"sleepy-hotstuff/src/config"
	"sleepy-hotstuff/src/cryptolib"
	"sleepy-hotstuff/src/db"
	"sleepy-hotstuff/src/logging"
	"sleepy-hotstuff/src/message"
	pb "sleepy-hotstuff/src/proto/communication"
	"sleepy-hotstuff/src/quorum"
	"sleepy-hotstuff/src/utils"
	"strconv"
	"sync"
	"time"
)

/*all the parameter for hotstuff protocols*/
var Sequence utils.IntValue  //current Sequence number
var curBlock message.QCBlock //recently voted block
var votedBlocks utils.IntByteMap
var lockedBlock message.QCBlock //locked block
var curHash utils.ByteValue     //current hash

// it seems that awaitingDecision and awaitingDecisionCopy are almost only written and not read.
// awaitingBlocks is read to fill the preHash and prepreHash,
// but will be cleared in view change. (if no view change, it will keep growing).
// So it seems that the first block of a new view will not get its preHash and prepreHash filled.
var awaitingBlocks utils.IntByteMap
var awaitingBlocksTXS utils.IntBytesMap
var awaitingDecision utils.IntByteMap
var awaitingDecisionCopy utils.IntByteMap

var committedBlocks utils.IntByteMap // record the committed block history.

var vcAwaitingVotes utils.IntIntMap
var vcTime int64

var timer *time.Timer // timer for view changes

var cblock sync.Mutex
var lqcLock sync.RWMutex
var bufferLock sync.Mutex

// A replica will only send out messages of a step if it enters the previous step
var buffer utils.StringIntMap

// evaluation
var curOPS utils.IntValue
var totalOPS utils.IntValue
var beginTime int64
var lastTime int64
var clock sync.Mutex
var genesisTime int64

var forcePrint bool

func InitHotStuff(thisid int64) {
	buffer.Init()

	t, p := config.FetchTestTypeAndParam()
	if t == config.Test_off {
		quorum.StartQuorum(n)
	} else {
		InTestConfig(t, p)
	}

	Sequence.Init()
	InitView()
	db.PersistValue("Sequence", &Sequence, db.PersistAll)
	votedBlocks.Init()
	db.PersistValue("votedBlocks", &votedBlocks, db.PersistAll)
	awaitingBlocks.Init()
	awaitingBlocksTXS.Init()
	db.PersistValue("awaitingBlocks", &awaitingBlocks, db.PersistAll)
	awaitingDecision.Init()
	db.PersistValue("awaitingDecision", &awaitingDecision, db.PersistAll)
	awaitingDecisionCopy.Init()
	db.PersistValue("awaitingDecisionCopy", &awaitingDecisionCopy, db.PersistAll)

	if committedBlocks.GetLen() == 0 {
		committedBlocks.Init()
		db.PersistValue("committedBlocks", &committedBlocks, db.PersistCritical)
	}
	SetView(0)

	timeoutBuffer.Init(n)
	recBuffer.Init()

	curBlock = message.QCBlock{}
	lockedBlock = message.QCBlock{}
	curHash.Init()
	vcAwaitingVotes.Init()

	cryptolib.StartECDSA(thisid)

	forcePrint = true

	//if config.EvalMode() > 0 {
	//	curOPS.Init()
	//	totalOPS.Init()
	//}
}

func getTransactions(batch []pb.RawMessage) [][]byte {
	txs := make([][]byte, len(batch))
	for i := 0; i < len(batch); i++ {
		txs[i] = batch[i].GetMsg() // message.MessageWithSignature
	}
	return txs
}

// This func is invoked by the leader to broadcast a new proposal,
// so it may be invoked for many times by one node.
func StartHotStuff(batch []pb.RawMessage) {
	seq := Increment()
	msg := message.HotStuffMessage{
		Mtype:  pb.MessageType_QC,
		Seq:    seq,
		Source: id,
		View:   LocalView(),
		OPS:    batch,
		TS:     utils.MakeTimestamp(),
		Num:    quorum.NSize(),
	}

	msg.QC = FetchBlockInfo(seq)
	//if curBlock.Height != 0 {
	//	msg.Seq = curBlock.Height + 1
	//}
	// It seems that Seq is not only the height of block, but also an identifier of
	//the hotstuff message.
	// Unfortunately, the Seq is also used as the height of block, which are bugs needed to correct.
	tmphash := msg.GetMsgHash()
	msg.Hash = ObtainCurHash(tmphash, seq) // Hash(curHash + Hash(seq+OPS))
	curHash.Set(msg.Hash)
	db.PersistValue("curHash", &curHash, db.PersistAll)
	awaitingBlocks.Insert(seq, msg.Hash)
	txs := getTransactions(batch)
	awaitingBlocksTXS.SetValue(seq, txs)
	db.PersistValue("awaitingBlocks", &awaitingBlocks, db.PersistAll)
	awaitingDecisionCopy.Insert(seq, msg.Hash)
	db.PersistValue("awaitingDecisionCopy", &awaitingDecisionCopy, db.PersistAll)

	log.Printf("proposing block with height %d, awaiting %d blocks", seq, awaitingDecisionCopy.GetLen())

	msgbyte, _ := msg.Serialize()
	request, _ := message.SerializeWithSignature(id, msgbyte)
	go HandleQCByteMsg(request) // this message is first ``received'' by the node itself.
	sender.RBCByteBroadcast(msgbyte)
}

// Fetch the current block (can be used as the parent block).
// The seq inputted is only used to check whether this is the first block.
func FetchBlockInfo(seq int) []byte {
	if seq == 1 || curBlock.Height == 0 {
		return nil //return nil, representing initial block
	}
	var msg []byte
	var err error
	cblock.Lock()
	msg, err = curBlock.Serialize()
	cblock.Unlock()
	if err != nil {
		log.Printf("fail to serialize curblock")
		return []byte("")
	}

	return msg
}

func GetSeq() int {
	return Sequence.Get()
}

func Increment() int {
	Sequence.Increment()
	db.PersistValue("Sequence", &Sequence, db.PersistAll)
	return Sequence.Get()
}

func UpdateSeq(seq int) {
	if seq > GetSeq() {
		Sequence.Set(seq)
		db.PersistValue("Sequence", &Sequence, db.PersistAll)
		// log.Printf("update sequence to %v", Sequence.Get())
	}
}

func GenHashOfTwoVal(input1 []byte, input2 []byte) []byte {
	tmp := make([][]byte, 2, 2)
	tmp[0] = input1
	tmp[1] = input2
	b := bytes.Join(tmp, []byte(""))
	return cryptolib.GenHash(b)
}

func ObtainCurHash(input []byte, seq int) []byte {
	var result []byte
	bh := GenHashOfTwoVal(utils.IntToBytes(seq), input)
	ch := curHash.Get()
	result = GenHashOfTwoVal(ch, bh)
	return result
}

/*
Get data from buffer and cache. Used for consensus status
*/
func GetBufferContent(key string, btype TypeOfBuffer) (ConsensusStatus, bool) {
	switch btype {
	case BUFFER:
		v, exist := buffer.Get(key)
		return ConsensusStatus(v), exist
	}
	return 0, false
}

/*
Update buffer and cache. Used for consensus status
*/
func UpdateBufferContent(key string, value ConsensusStatus, btype TypeOfBuffer) {
	switch btype {
	case BUFFER:
		buffer.Insert(key, int(value))
	}
}

/*
Delete buffer and cache. Used for consensus status
*/
func DeleteBuffer(key string, btype TypeOfBuffer) {
	switch btype {
	case BUFFER:
		buffer.Delete(key)
	}
}

func HandleQCByteMsg(inputMsg []byte) {
	tmp := message.DeserializeMessageWithSignature(inputMsg)
	input := tmp.Msg
	//TODO: tmp.Sig should be verified.

	content := message.DeserializeHotStuffMessage(input)

	mtype := content.Mtype
	source := content.Source
	communication.SetLive(utils.Int64ToString(source))

	// log.Printf("receive a %v msg from replica: %v at seq: %d", mtype, source, content.Seq)

	sleepLock.RLock()
	defer sleepLock.RUnlock()
	if curStatus.Get() == SLEEPING {
		return
	}
	if curStatus.Get() == RECOVERING {
		if mtype != pb.MessageType_ECHO1 && mtype != pb.MessageType_ECHO2 && mtype != pb.MessageType_TQC {
			return
		}
	}

	switch mtype {
	case pb.MessageType_QC:
		HandleNormalMsg(content)
	case pb.MessageType_QCREP:
		HandleNormalRepMsg(content)
	case pb.MessageType_TIMEOUT:
		HandleTimeoutMsg(content, tmp)
	case pb.MessageType_TQC:
		HandleTQCMsg(content)
	case pb.MessageType_VIEWCHANGE:
		HandleQCVCMessage(content, tmp)
	case pb.MessageType_NEWVIEW:
		HandleQCNewView(tmp.Msg)
	case pb.MessageType_REC1:
		HandleRec1Msg(content)
	case pb.MessageType_ECHO1:
		HandleEcho1Msg(content)
	case pb.MessageType_REC2:
		HandleRec2Msg(content)
	case pb.MessageType_ECHO2:
		HandleEcho2Msg(content)
	}
}

func rank(block message.QCBlock, blocktwo message.QCBlock) int {
	//log.Printf("blockview %v, two view %v, height %v, %v", block.View, blocktwo.View, block.Height, blocktwo.Height)
	if block.View < blocktwo.View {
		return -1
	}
	if block.Height < blocktwo.Height {
		return -1
	} else if block.Height == blocktwo.Height {
		return 0
	}
	return 1
}

// comments: TODO: blockinfo should be verified that its height value is correct. (from genisis to it)
func VerifyBlock(curSeq int, source int64, blockinfo message.QCBlock) bool {
	if curSeq == 1 && (Sequence.Get() < curSeq || source == id) {
		return true
	}

	if blockinfo.Height < lockedBlock.Height {
		return false
	}

	if len(blockinfo.QC) < quorum.QuorumSize() { // blockinfo.View != LocalView()||
		return false
	}
	if !VerifyQC(blockinfo) {
		p := fmt.Sprintf("[QC] block signature %d not verified", blockinfo.Height)
		logging.PrintLog(true, logging.ErrorLog, p)
		return false
	}
	return true
}

func HandleNormalMsg(content message.HotStuffMessage) { //For replica to process proposals from the leader
	viewMux.RLock()
	defer viewMux.RUnlock()
	if content.View < LocalView() || curStatus.Get() == VIEWCHANGE {
		return
	}

	hash := ""
	source := content.Source

	hash = utils.BytesToString(content.Hash)
	if config.EvalMode() > 0 {
		evaluation(len(content.OPS), content.Seq)
	}

	if vcTime > 0 {
		// this seems not to be ture in view 0,
		//since vcTime is not assigned a value in view 0.
		vcdTime := utils.MakeTimestamp()
		log.Printf("processing block sequence %v, %v ms", content.Seq, vcdTime-vcTime)
	}

	if content.OPS != nil {
		awaitingDecision.Insert(content.Seq, content.Hash)
		db.PersistValue("awaitingDecision", &awaitingDecision, db.PersistAll)
		//dTime := utils.MakeTimestamp()
		//diff,_ := utils.Int64ToInt(dTime - cTime)
		//log.Printf("[%v] ++latency-1 for QCM %v ms", content.Seq, diff)
		if !Leader() {
			awaitingDecisionCopy.Insert(content.Seq, content.Hash)
			db.PersistValue("awaitingDecisionCopy", &awaitingDecisionCopy, db.PersistAll)
		}
	}
	// TODO： content.Seq should also be checked whether it has been voted.
	blockinfo := message.DeserializeQCBlock(content.QC)
	if !VerifyBlock(content.Seq, content.Source, blockinfo) {
		log.Printf("[QC] HotStuff Block with height %d not verified", blockinfo.Height)
		p := fmt.Sprintf("[QC] HotStuff Block %d not verified", blockinfo.Height)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}
	/*if content.OPS != nil{
		dTime := utils.MakeTimestamp()
		diff,_ := utils.Int64ToInt(dTime - cTime)
		log.Printf("[%v] ++latency-2 for QCM %v ms", content.Seq, diff)
	}*/
	p := fmt.Sprintf("[QC] HotStuff processing QC proposed at height %d", content.Seq)
	logging.PrintLog(verbose, logging.NormalLog, p)

	ProcessQCInfo(hash, blockinfo, content)
	msg := message.HotStuffMessage{
		Mtype:  pb.MessageType_QCREP,
		Source: id,
		View:   LocalView(),
		Hash:   content.Hash,
		Seq:    content.Seq,
	}

	sig := cryptolib.GenSig(id, content.Hash)
	msg.Sig = sig // a signature for the voted block, not for the entire message

	if !cryptolib.VerifySig(id, content.Hash, msg.Sig) {
		p := fmt.Sprintf("%d can not verify its newly generated sig!", id)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	msgbyte, err := msg.Serialize()
	if err != nil {
		logging.PrintLog(true, logging.ErrorLog, "[QCMessage Error] Not able to serialize the message")
		return
	}
	msgwithsig, _ := message.SerializeWithSignature(id, msgbyte)
	if Leader() {
		// the message is first received by the leader itself.
		go HandleQCByteMsg(msgwithsig)
	}
	go sender.SendToNode(msgbyte, source, message.HotStuff)
}

// it seems that this func is useless, since queueHead is not set to a value in another place.
func HandleQueue(ch string, ops []pb.RawMessage) {
	if Leader() {
		return
	}
	if queueHead.Get() != "" {
		//log.Printf("stop queue head %s", queueHead)
		queueHead.Set("")
		timer.Stop()
	}
}

func outputBlockchain(height int, blockchain utils.IntByteMap) error {
	if !forcePrint && height > 10 {
		return nil
	}
	forcePrint = false

	type TX struct {
		ID        int64
		TX        message.Transaction
		Timestamp int64
	}
	type block struct {
		View   int
		Height int
		// Hash    []byte
		// PreHash []byte
		TXS []TX
	}
	// print out genesis block
	btx := TX{
		ID: 0,
		TX: message.Transaction{
			From:  "",
			To:    "0",
			Value: 50,
			// Timestamp: time.Now().UnixNano(),
		},
		Timestamp: 0,
	}
	txs := make([]TX, 1)
	txs[0] = btx
	gb := block{
		View:   0,
		Height: 0,
		// Hash:    nil,
		// PreHash: nil,
		TXS: txs,
	}
	jsonData, err := json.Marshal(gb)
	if err != nil {
		return err
	}
	log.Println(string(jsonData))

	tsmap := utils.IntBoolMap{}
	tsmap.Init()

	for i := 1; i <= height; i++ {
		bser, exist := blockchain.Get(i)
		if !exist {
			//log.Printf("the length of blockchain is: %d", blockchain.GetLen())
			//err = errors.New("no block at height " + strconv.Itoa(i))
			//log.Printf("%v", err)
			continue
		}
		b := message.DeserializeQCBlock(bser)
		txs = make([]TX, 0)
		for j := 0; j < len(b.TXS); j++ {
			var cr message.ClientRequest
			cr = message.DeserializeClientRequest(b.TXS[j].Msg)
			_, ex := tsmap.Get(int(cr.TS))
			if ex {
				// skip repeated transactions. (the timestamp serves as the unique identifier)
				continue
			}
			tsmap.Insert(int(cr.TS), true)
			var tx TX
			tx.ID = cr.ID
			err = tx.TX.Deserialize(cr.OP)
			if err != nil {
				log.Printf("Error: height:%d, OP: %v", i, cr.OP)
				tx.TX = message.Transaction{}
				// log.Fatal(err)
			}
			// fmt.Println(txs[j].TX)
			tx.Timestamp = cr.TS
			txs = append(txs, tx)
		}
		bb := block{
			View:   b.View,
			Height: b.Height,
			// Hash:    b.Hash,
			// PreHash: b.PreHash,
			TXS: txs,
		}
		jsonData, err = json.Marshal(bb)
		if err != nil {
			return err
		}
		log.Println(string(jsonData))
	}
	return nil
}

func ProcessQCInfo(hash string, blockinfo message.QCBlock, content message.HotStuffMessage) {
	if blockinfo.Height >= 2 {
		if blockinfo.Height <= curBlock.Height {
			return
		}
		if blockinfo.Height >= 3 {
			// deliver/commit block
			lqcLock.RLock()
			blockser, _ := lockedBlock.Serialize()
			if blockinfo.Height >= lockedBlock.Height+2 {
				// Bug: for the leader, blockinfo.height = lockedBlock.height+1,
				//so the leader cannot insert blocks into committedblocks.
				committedBlocks.Insert(lockedBlock.Height, blockser)
				log.Printf("[!!!] Ready to output a value for height %d", lockedBlock.Height)
				if testid, _ := config.FetchTestTypeAndParam(); testid == config.Test_Koala2_DoubleSpend ||
					testid == config.Test_HotStuff_NoPersist_DoubleSpend ||
					testid == config.Test_HotStuff_Persist_DoubleSpend {
					err := outputBlockchain(lockedBlock.Height, committedBlocks) // print out the blockchain.
					if err != nil {
						log.Printf("[!!!] Error processing block seq %d: %v", content.Seq, err)
					}
				}
			}
			// log.Printf("blockinfo height: %d, lockedblock height: %d, curBlock height: %d", blockinfo.Height, lockedBlock.Height, curBlock.Height)
			lqcLock.RUnlock()
			db.PersistValue("committedBlocks", &committedBlocks, db.PersistCritical)
		}
		lqcLock.Lock()
		lockedBlock = curBlock // why curBlock is exactly blockinfo's parent?
		// todo: for leader, this lockedBlock is wrongly updated to the prepared block.
		// This is because the curBlock has been updated to blockinfo.
		// The needed modification may be complex, since we need to set the lockedblock to
		//the exact parent block of curblock when updating curblock.
		db.PersistValue("lockedBlock", &lockedBlock, db.PersistCritical)
		lqcLock.Unlock()
		votedBlocks.Delete(curBlock.Height)
		db.PersistValue("votedBlocks", &votedBlocks, db.PersistAll)
	}
	if !Leader() && blockinfo.Height > curBlock.Height {
		cblock.Lock()
		curBlock = blockinfo
		db.PersistValue("curBlock", &curBlock, db.PersistAll)
		cblock.Unlock()
		curHash.Set(curBlock.Hash)
		db.PersistValue("curHash", &curHash, db.PersistAll)
	}

	if content.Seq > 3 {
		//awaitingBlocks.Delete(content.Seq-3)
		awaitingDecision.Delete(content.Seq - 3)
		db.PersistValue("awaitingDecision", &awaitingDecision, db.PersistAll)
		awaitingDecisionCopy.Delete(content.Seq - 3)
		db.PersistValue("awaitingDecisionCopy", &awaitingDecisionCopy, db.PersistAll)
	}

	if curBlock.PrePreHash != nil {

		ch := utils.BytesToString(curBlock.PrePreHash)
		p := fmt.Sprintf("[QC] HotStuff deliver block at height %d", content.Seq-3)
		logging.PrintLog(verbose, logging.NormalLog, p)

		/*if vcTime >0 {
			vcdTime := utils.MakeTimestamp()
			log.Printf("deliver block height %v, %v ms", content.Seq-3, vcdTime - vcTime)
		}*/
		//log.Printf("***[%v] deliver request, curSeq %v", content.Seq-3, sequence.GetSeq())
		if config.EvalMode() == 0 {
			//todo deliver block
		} else {

		}

		if content.OPS != nil {
			go HandleQueue(ch, content.OPS)
		}
	}
	UpdateSeq(content.Seq)
}

func HandleNormalRepMsg(content message.HotStuffMessage) {
	viewMux.RLock()
	defer viewMux.RUnlock()
	if content.View < LocalView() || curStatus.Get() == VIEWCHANGE {
		return
	}

	h, exist := awaitingBlocks.Get(content.Seq)
	// the block of Seq must be the one 'I' proposed
	if exist && bytes.Compare(h, content.Hash) != 0 {
		p := fmt.Sprintf("[QC] hash not matching", content.Seq)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}
	if !cryptolib.VerifySig(content.Source, content.Hash, content.Sig) {
		p := fmt.Sprintf("[QC] signature for QCRep with height %v not verified", content.Seq)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}
	hash := utils.BytesToString(content.Hash)

	bufferLock.Lock()
	defer bufferLock.Unlock()
	// check that if there has existed a prepare qc for the block of hash.
	v, _ := GetBufferContent("BLOCK"+hash, BUFFER)
	if v == PREPARED {
		// log.Printf("There has been a prepare QC for content.Seq: %v", content.Seq)
		return
	}

	quorum.Add(content.Source, hash, content.Sig, quorum.PP)
	if quorum.CheckQuorum(hash, quorum.PP) {
		UpdateBufferContent("BLOCK"+hash, PREPARED, BUFFER)

		cer_byte := quorum.FetchCer(hash)
		if cer_byte == nil {
			p := fmt.Sprintf("[QC] cannnot obtain certificate from cache for block %v", content.Seq)
			logging.PrintLog(verbose, logging.ErrorLog, p)
		}
		_, sigs, ids := message.DeserializeSignatures(cer_byte)

		qcblock := message.QCBlock{
			View: LocalView(),
			// Height: content.Seq,
			Height: curBlock.Height + 1, // this is a temporary solution,
			// since curBlock might not be the parent block of this qcblock (I'm not sure).
			QC:   sigs,
			IDs:  ids,
			Hash: content.Hash,
		}

		ph, any := awaitingBlocks.Get(content.Seq - 1)
		if any && ph != nil {
			qcblock.PreHash = ph
		}
		ph, any = awaitingBlocks.Get(content.Seq - 2)
		if any && ph != nil {
			qcblock.PrePreHash = ph
		}

		var txs [][]byte
		txs, any = awaitingBlocksTXS.Get(content.Seq)
		if any && txs != nil {
			qcblock.TXS = make([]message.MessageWithSignature, len(txs)+1)
			btx := message.Transaction{
				From:  "",
				To:    strconv.Itoa(int(id)),
				Value: 50,
				// Timestamp: time.Now().UnixNano(),
			}
			btxser, _ := btx.Serialize()
			cr := message.ClientRequest{
				ID: id,
				OP: btxser,
				TS: utils.MakeTimestamp(),
			}
			crser, _ := cr.Serialize()
			qcblock.TXS[0] = message.MessageWithSignature{
				Msg: crser,
			}
			for i := 1; i < len(txs)+1; i++ {
				qcblock.TXS[i] = message.DeserializeMessageWithSignature(txs[i-1])
			}
		}

		if qcblock.Height > curBlock.Height {
			cblock.Lock()
			curBlock = qcblock
			db.PersistValue("curBlock", &curBlock, db.PersistAll)
			cblock.Unlock()
		} else {
			log.Printf("New block's QC has Seq: %v <= curBlock.Height: %v", content.Seq, curBlock.Height)
		}
		curStatus.Set(READY)
	}
}

// Get the throughput of the system
func evaluation(lenOPS int, seq int) {
	if lenOPS > 1 {
		//log.Printf("[Replica] evaluation mode with %d ops", lenOPS)
		var p = fmt.Sprintf("[Replica] evaluation mode with %d ops", lenOPS)
		logging.PrintLog(false, logging.EvaluationLog, p)
	}

	clock.Lock()
	defer clock.Unlock()
	val := curOPS.Get()
	if seq == 1 {
		beginTime = utils.MakeTimestamp()
		lastTime = beginTime
		genesisTime = utils.MakeTimestamp()
	}

	tval := totalOPS.Get()
	lenOPS = lenOPS //* 5
	tval = tval + lenOPS
	totalOPS.Set(tval)

	// the logic is wrong.
	// Here, the throughput is coumputed by the number of operations in the new block
	//deviding the latency of the last block.
	// It can only be correct when the number of operations of every block is the same.
	// Besides, the 'latency' is not the commit latency.
	// The commit latency of block i should be latency(i+3)+latency(i+2)+latency(i+1)+
	//the time of the leader packing and sending the block i, where latency(i) is the evaluated value.
	if val+lenOPS >= config.MaxBatchSize() {
		curOPS.Set(0)
		var endTime = utils.MakeTimestamp()
		var throughput int
		lat, _ := utils.Int64ToInt(endTime - beginTime)
		if lat > 0 {
			throughput = 1000 * (val + lenOPS) / lat // tx/s
		}

		clockTime, _ := utils.Int64ToInt(utils.MakeTimestamp() - genesisTime)
		log.Printf("[Replica] Processed %d (ops=%d, clockTime=%d ms, seq=%v) operations using %d ms. "+
			"Throughput %d tx/s. ", tval, lenOPS, clockTime, seq, lat, throughput)
		var p = fmt.Sprintf("[Replica] Processed %d (ops=%d, clockTime=%d ms, seq=%v) operations using %d ms. "+
			"Throughput %d tx/s", tval, lenOPS, clockTime, seq, lat, throughput)
		logging.PrintLog(true, logging.EvaluationLog, p)
		beginTime = endTime
		lastTime = beginTime
	} else {
		curOPS.Set(val + lenOPS)
	}
}
