package db

import (
	"errors"
	"fmt"
	"log"
	"sleepy-hotstuff/src/config"
	"sleepy-hotstuff/src/logging"
)

type PersistLevelType int

const (
	PersistAll      PersistLevelType = 1
	PersistCritical PersistLevelType = 2
	NoPersist       PersistLevelType = 3
)

type DBValue interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error // The variable itself must contain the result of deserialization
}

// hotstuff

// var Sequence utils.IntValue  //current Sequence number
// var curBlock message.QCBlock //recently voted block
// var votedBlocks utils.IntByteMap
// var lockedBlock message.QCBlock //locked block
// var curHash utils.ByteValue     //current hash
// var awaitingBlocks utils.IntByteMap
// var awaitingDecision utils.IntByteMap
// var awaitingDecisionCopy utils.IntByteMap
// var vcAwaitingVotes utils.IntIntMap

// viewchange
// var view int

// event
// var queue consensus.Queue

//messages
// var msgQueue consensus.Queue

// delivered blocks
// var committedBlocks utils.IntByteMap

func PersistValue(key string, value DBValue, level PersistLevelType) {
	if PersistLevelType(config.PersistLevel()) <= level {
		msg := fmt.Sprintf("store value in database: %s, persist level: %d", key, level)
		logging.PrintLog(false, logging.NormalLog, msg)
		// time.Sleep(100 * time.Millisecond)
		err := WriteDB(key, value)
		if err != nil {
			log.Fatalf("error writing Sequence to database: %v", err)
		}
	}
}

func RecoverValue(key string, value DBValue) error {
	plevel := PersistLevelType(config.PersistLevel())
	if plevel == NoPersist {
		msg := fmt.Sprintf("No value in database: persist level: NoPersist")
		return errors.New(msg)
	}
	if plevel == PersistCritical {
		if key == "committedBlocks" || key == "view" || key == "lockedBlock" {
			err := ReadDB(key, value)
			return err
		} else {
			msg := fmt.Sprintf("The %s is not stored in the database for the persistLevel %", key, plevel)
			return errors.New(msg)
		}
	}
	if plevel == PersistAll {
		err := ReadDB(key, value)
		return err
	} else {
		msg := fmt.Sprintf("The PersistLevelType in config: %s is not planned.", plevel)
		return errors.New(msg)
	}
}
