package query

import "testing"

func TestRiskScore(t *testing.T) {
	cases := []struct {
		name string
		in   RiskInputs
		want int
	}{
		{"none", RiskInputs{}, 0},
		{"exported_only", RiskInputs{IsExported: true}, 2},
		{"hub_only", RiskInputs{InHub: true}, 3},
		{"interface_change_only", RiskInputs{InterfaceChange: true}, 3},
		{"untested_only", RiskInputs{Untested: true}, 1},
		{"all_factors", RiskInputs{IsExported: true, InHub: true, InterfaceChange: true, Untested: true}, 9},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RiskScore(c.in); got != c.want {
				t.Errorf("got=%d want=%d", got, c.want)
			}
		})
	}
}
