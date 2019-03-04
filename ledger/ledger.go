package ledger

import (
	"github.com/google/uuid"
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
	ledger.Set(scn.UUID.URN(), LedgerEntry{State: "DECLARED", Scenario: scn})
}

func Unregister(uuid uuid.UUID) {
	ledger.Remove(uuid.URN())
}

func Count() int {
	return ledger.Count()
}
