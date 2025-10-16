package consensus

import (
	"log"
	"sleepy-hotstuff/src/communication/sender"
	"sleepy-hotstuff/src/config"
	"sleepy-hotstuff/src/cryptolib"
	"sleepy-hotstuff/src/db"
	"sleepy-hotstuff/src/utils"
	"sync"
)

type ConsensusType int

const (
	HotStuff ConsensusType = 2
)

type RbcType int

const (
	RBC   RbcType = 0
	ECRBC RbcType = 1
)

type TypeOfBuffer int32

const (
	BUFFER TypeOfBuffer = 0
	CACHE  TypeOfBuffer = 1
)

type ConsensusStatus int

const (
	PREPREPARED ConsensusStatus = 0
	PREPARED    ConsensusStatus = 1
	COMMITTED   ConsensusStatus = 2
	BOTHPANDC   ConsensusStatus = 3
)

func GetInstanceID(input int) int {
	return input + n*epoch.Get() //baseinstance*epoch.Get()
}

func GetIndexFromInstanceID(input int, e int) int {
	return input - n*e
}

func GetInstanceIDsOfEpoch() []int {
	var output []int
	for i := 0; i < len(members); i++ {
		output = append(output, GetInstanceID(members[i]))
	}
	return output
}

func StartHandler(rid string, mem string) {
	id, errs = utils.StringToInt64(rid)
	if errs != nil {
		log.Printf("[Error] Replica id %v is not valid. Double check the configuration file", id)
		return
	}
	iid, _ = utils.StringToInt(rid)

	config.LoadConfig()
	cryptolib.StartCrypto(id, config.CryptoOption())
	consensus = ConsensusType(config.Consensus())

	n = config.FetchNumReplicas()
	curStatus.Init()
	epoch.Init()
	midTime = make(map[int]int64)
	queue.Init()
	db.PersistValue("queue", &queue, db.PersistAll)
	MsgQueue.Init()
	db.PersistValue("queue", &MsgQueue, db.PersistAll)
	verbose = config.FetchVerbose()
	sleepTimerValue = config.FetchSleepTimer()

	log.Printf("sleeptimer value %v", sleepTimerValue)
	switch consensus {
	case HotStuff:
		log.Printf("running HotStuff")
		InitHotStuff(id)
		if config.EvalMode() > 0 {
			// genesisTime = utils.MakeTimestamp()
			curOPS.Init()
			totalOPS.Init()
		}
	default:
		log.Fatalf("Consensus type not supported")
	}

	sender.StartSender(rid)
	go RequestMonitor(LocalView())

	//go func() {
	//	for true {
	//		log.Printf("in for print!")
	//		runtime.Gosched()
	//	}
	//}()

	if t, _ := config.FetchTestTypeAndParam(); t != config.Test_off {
		go TestSleepAndRecover(id)
	}
}

type QueueHead struct {
	Head string
	sync.RWMutex
}

func (c *QueueHead) Set(head string) {
	c.Lock()
	defer c.Unlock()
	c.Head = head
}

func (c *QueueHead) Get() string {
	c.RLock()
	defer c.RUnlock()
	return c.Head
}

type CurStatus struct {
	enum Status
	sync.RWMutex
}

type Status int

const (
	READY      Status = 0
	PROCESSING Status = 1
	VIEWCHANGE Status = 2
	SLEEPING   Status = 3
	RECOVERING Status = 4
)

func (c *CurStatus) Set(status Status) {
	c.Lock()
	defer c.Unlock()
	c.enum = status
}

func (c *CurStatus) Init() {
	c.Lock()
	defer c.Unlock()
	c.enum = READY
}

func (c *CurStatus) Get() Status {
	c.RLock()
	defer c.RUnlock()
	return c.enum
}
