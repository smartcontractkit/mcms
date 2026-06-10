package canton

// LedgerQueryParties builds the party list for ledger ACS reads (GetActiveContracts, etc.).
//
// The operator participant may ActAs one party while holding CanReadAs for others (for example
// CCIP owner parties under MCMS). Contract visibility is per-party: an MCMS instance might only
// be readable via ReadAs party B even when ActAs party A cannot see it.
//
// Order is ActAs party first, then ReadAs parties in config order. Duplicates and empty strings
// are omitted.
func LedgerQueryParties(participant Participant) []string {
	parties := make([]string, 0, 1+len(participant.ReadAsPartyIDs))
	seen := make(map[string]struct{}, 1+len(participant.ReadAsPartyIDs))
	add := func(party string) {
		if party == "" {
			return
		}
		if _, ok := seen[party]; ok {
			return
		}
		seen[party] = struct{}{}
		parties = append(parties, party)
	}
	add(participant.PartyID)
	for _, party := range participant.ReadAsPartyIDs {
		add(party)
	}

	return parties
}

// MCMSPartiesForChain returns the deduplicated party list for MCMS ledger queries across all
// chain participants.
func MCMSPartiesForChain(ch Chain) []string {
	parties := make([]string, 0)
	seen := make(map[string]struct{})
	for _, participant := range ch.Participants {
		for _, party := range LedgerQueryParties(participant) {
			if _, ok := seen[party]; ok {
				continue
			}
			seen[party] = struct{}{}
			parties = append(parties, party)
		}
	}

	return parties
}
