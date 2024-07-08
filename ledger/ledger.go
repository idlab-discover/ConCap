// Package ledger is a very basic start to do bookkeeping in a concurrent map of experiment / pod states
// This may be scrapped entirely in favor of standard kubernetes tooling
// It currently has impact on the capturing functionality of containercap (gocap)
package ledger

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ledger sync.Map

type ScenarioState string

const (
	SCHEDULED ScenarioState = "scheduled"
	STARTING  ScenarioState = "starting"
	RUNNING   ScenarioState = "running"
	COMPLETED ScenarioState = "completed"
	ERROR     ScenarioState = "error"
	NO_STATE  ScenarioState = "no state"
)

type LedgerEntry struct {
	State ScenarioState
	Time  time.Time
}

func Register(uuid uuid.UUID) {
	ledger.Store(uuid, LedgerEntry{State: SCHEDULED, Time: time.Now()})
	log.Println(uuid, GetScenarioState(uuid))
}

func UpdateState(uuid uuid.UUID, le LedgerEntry) {
	ledger.Store(uuid, le)
	log.Println("Ledger - Scenario: " + GetScenarioState(uuid))
}

func Unregister(uuid uuid.UUID) {
	ledger.Delete(uuid)
}

func GetScenarioState(uuid uuid.UUID) ScenarioState {

	var l interface{}
	l, _ = ledger.Load(uuid)
	if l != nil {
		return l.(LedgerEntry).State
	}
	return NO_STATE
}
