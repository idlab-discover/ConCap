// Package ledger is a very basic start to do bookkeeping in a concurrent map of experiment / pod states
// This may be scrapped entirely in favor of standard kubernetes tooling
// It currently has no impact on the functionality of containercap
// The objective is to access / use the ledger from a simple terminal UI
package ledger

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/k0kubun/pp"
	cmap "github.com/orcaman/concurrent-map"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

var ledger cmap.ConcurrentMap

type LedgerEntry struct {
	State    string
	Scenario *scenario.Scenario
}

func init() {
	ledger = cmap.New()
}

func Register(scn *scenario.Scenario) {
	ledger.Set(scn.UUID.String(), LedgerEntry{State: "DECLARED", Scenario: scn})
}

func UpdateState(uuid string, le LedgerEntry) {
	ledger.Set(uuid, le)
}

func Unregister(uuid uuid.UUID) {
	ledger.Remove(uuid.String())
}

func Count() int {
	return ledger.Count()
}

func Keys() []string {
	return ledger.Keys()
}

func Dump() {
	for i := range ledger.IterBuffered() {
		pp.Printf("%+v\n", i)
	}
}

func Repr() {
	for i := range ledger.IterBuffered() {
		fmt.Printf("%+v\n", i)
	}
}
