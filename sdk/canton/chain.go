package canton

import (
	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
)

// LedgerServices holds the ledger API clients required by MCMS Canton integrations.
type LedgerServices struct {
	State   apiv2.StateServiceClient
	Command apiv2.CommandServiceClient
}

// Participant is a Canton ledger participant used by MCMS.
type Participant struct {
	PartyID        string
	LedgerServices LedgerServices
}

// Chain holds Canton participants for a chain selector.
// Callers that use chainlink-deployments-framework should map cldf canton.Chain to this type
// when implementing chainwrappers.ChainAccessor.
type Chain struct {
	Participants []Participant
}
