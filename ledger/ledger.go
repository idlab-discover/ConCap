// Package ledger is a very basic start to do bookkeeping in a concurrent map of experiment / pod states
// This may be scrapped entirely in favor of standard kubernetes tooling
// It currently has no impact on the functionality of containercap
// The objective is to access / use the ledger from a simple terminal UI
package ledger

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ledger sync.Map

type ScenarioState string

const (
	DECLARED  ScenarioState = "declared"
	STARTING  ScenarioState = "starting"
	RUNNING   ScenarioState = "running"
	COMPLETED ScenarioState = "completed"
	ERROR     ScenarioState = "error"
	BUNDLED   ScenarioState = "bundled"
)

type LedgerEntry struct {
	State ScenarioState
	Time  time.Time
}

func Register(uuid string) {
	ledger.Store(uuid, LedgerEntry{State: DECLARED, Time: time.Now()})
	fmt.Println(uuid, GetScenarioState(uuid))
}

func UpdateState(uuid string, le LedgerEntry) {
	ledger.Store(uuid, le)
	fmt.Println("Ledger - Scenario: " + GetScenarioState(uuid))
}

func Unregister(uuid uuid.UUID) {
	ledger.Delete(uuid.String())
}

func GetScenarioState(uuid string) string {

	var l interface{}
	l, _ = ledger.Load(uuid)
	if l != nil {
		return string(l.(LedgerEntry).State)
	}
	return "no state"
}

func Keys() []string {
	m := []string{}
	ledger.Range(func(key interface{}, value interface{}) bool {
		m = append(m, fmt.Sprint(key))
		return true
	})
	return m
}
