package query

// RiskInputs are the four boolean factors that feed RiskScore. The formula
// matches §6 of the graph design doc: 2*exported + 3*hub + 3*iface + 1*untested.
type RiskInputs struct {
	IsExported      bool
	InHub           bool
	InterfaceChange bool
	Untested        bool
}

// RiskScore is the deterministic risk weight used by Preflight to rank
// changed symbols. Higher means higher review priority.
func RiskScore(in RiskInputs) int {
	score := 0
	if in.IsExported {
		score += 2
	}
	if in.InHub {
		score += 3
	}
	if in.InterfaceChange {
		score += 3
	}
	if in.Untested {
		score += 1
	}
	return score
}
