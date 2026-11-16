package roster

import "fmt"

// The guild's in-game ranks, indexed by rank number (0 is the guild master). Blizzard's
// roster API returns only the number, so the names live here and must follow any rank
// added, removed or renamed in game.
var guildRanks = []struct {
	name string
	alt  bool
}{
	{name: "Guild Master"},
	{name: "Officer"},
	{name: "Recruiter"},
	{name: "Officer Alt", alt: true},
	{name: "Veteran"},
	{name: "Alt", alt: true},
	{name: "Member"},
}

// rankName is nil for characters the sync has never seen, and "Rank N" for a rank
// added in game before this list caught up.
func rankName(rank *int) *string {
	if rank == nil {
		return nil
	}
	name := fmt.Sprintf("Rank %d", *rank)
	if *rank < len(guildRanks) {
		name = guildRanks[*rank].name
	}
	return &name
}

// isAltRank treats an unknown rank as a main, matching how new characters start.
func isAltRank(rank int) bool {
	return rank >= 0 && rank < len(guildRanks) && guildRanks[rank].alt
}
