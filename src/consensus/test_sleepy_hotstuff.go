package consensus

import (
	"log"
	"sleepy-hotstuff/src/config"
	"sleepy-hotstuff/src/quorum"
	"sleepy-hotstuff/src/utils"
	"time"
)

func InTestConfig(t config.TestType, p config.TestParam) {
	switch t {
	case config.Test_off:
		return
	case config.Test_SleepyHotStuff_PartChurn, config.Test_Koala2_DoubleSpend:
		conf, err := SleepyHotstuffConfig()
		if err != nil {
			log.Fatal(err)
		}
		quorum.StartSleepyHotstuffQuorum(n, f, s, conf)
	case config.Test_HotStuff_NoPersist_DoubleSpend:
		quorum.StartQuorum(n)
	case config.Test_HotStuff_Persist_DoubleSpend:
		quorum.StartQuorum(n)
	default:
	}
}

// check whether the replica 'id' is a sleepy replica.
func ParamOfSleepyReplica(id int64) (bool, config.SleepyReplica) {
	t, p := config.FetchTestTypeAndParam()
	switch t {
	case config.Test_off:
		return false, config.SleepyReplica{}
	case config.Test_SleepyHotStuff_PartChurn:
		// rule: select the last ss replicas to sleep.
		num := config.FetchNumReplicas()
		ss := p.NumOfActualSleep
		i, _ := utils.Int64ToInt(id)
		if i >= num-ss {
			return true, config.SleepyReplica{SleepTime: p.SleepTime,
				SleepSeq: p.SleepSeq, RecMode: config.RecKoala2}
		}
		return false, config.SleepyReplica{}
	case config.Test_HotStuff_NoPersist_DoubleSpend,
		config.Test_HotStuff_Persist_DoubleSpend,
		config.Test_Koala2_DoubleSpend:
		for i := 0; i < len(p.Replicas); i++ {
			if p.Replicas[i].Id == utils.Int64ToString(id) {
				return true, p.Replicas[i]
			}
		}
		return false, config.SleepyReplica{}

	default:
		return false, config.SleepyReplica{}
	}
}

func TestSleepAndRecover(input int64) {
	isSleepy, p := ParamOfSleepyReplica(input)
	if !isSleepy {
		return
	}
	t, _ := config.FetchTestTypeAndParam()
	switch t {
	case config.Test_HotStuff_NoPersist_DoubleSpend,
		config.Test_HotStuff_Persist_DoubleSpend,
		config.Test_SleepyHotStuff_PartChurn,
		config.Test_Koala2_DoubleSpend:
		// Sleep
		SleepProcess(p.SleepSeq, p.SleepTime)
		// Recover
		err := RecoveryProcess(p.RecMode)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func SleepProcess(sleepSeq int, sleepTime int) {
	for GetSeq() < sleepSeq {
		time.Sleep(1 * time.Nanosecond)
		// log.Printf("wait for seq: %d >= sleepSeq: %d", GetSeq(), sleepSeq)
		// runtime.Gosched()
	}
	log.Printf("Falling asleep in sequence %d...", sleepSeq)
	sleepLock.Lock()
	curStatus.Set(SLEEPING)
	sleepLock.Unlock()
	log.Printf("sleepTime: %d ms", sleepTime)
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	InitHotStuff(id)
	log.Printf("Wake up...")
}
