// Package ledger is a very basic start to do bookkeeping in a concurrent map of experiment / pod states
// This may be scrapped entirely in favor of standard kubernetes tooling
// It currently has no impact on the functionality of containercap
// The objective is to access / use the ledger from a simple terminal UI
package ledger

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

var ledger sync.Map

type ScenarioState string

const (
	DECLARED  ScenarioState = "declared"
	STARTING  ScenarioState = "starting"
	RUNNING   ScenarioState = "running"
	COMPLETED ScenarioState = "completed"
	ERROR     ScenarioState = "error"
)

type LedgerEntry struct {
	State    ScenarioState
	Scenario *scenario.Scenario
}

func Register(scn *scenario.Scenario) {
	ledger.Store(scn.UUID.String(), LedgerEntry{State: DECLARED, Scenario: scn})
}

func UpdateState(uuid string, le LedgerEntry) {
	ledger.Store(uuid, le)
}

func Unregister(uuid uuid.UUID) {
	ledger.Delete(uuid.String())
}

func Keys() []string {
	m := []string{}
	ledger.Range(func(key interface{}, value interface{}) bool {
		m = append(m, fmt.Sprint(key))
		return true
	})
	return m
}
