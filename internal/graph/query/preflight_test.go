package query

import (
	"reflect"
	"sort"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestCommunityFromPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"internal/payment/processor.go", "internal/payment"},
		{"api/checkout.go", "api"},
		{"cmd/devpilot/main.go", "cmd/devpilot"},
		{"main.go", ""},
		{"internal/a/b/c/d/e.go", "internal/a/b"}, // depth cap 3
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := communityFromPath(c.in); got != c.want {
				t.Errorf("got=%q want=%q", got, c.want)
			}
		})
	}
}

func TestEnrichChangedSymbol(t *testing.T) {
	nodes := []store.Node{
		{ID: "internal/payment/p.go::Charge", Kind: "method", Path: "internal/payment/p.go",
			Name: "Charge", Container: "PaymentProcessor", Language: "go", IsExported: true},
		{ID: "api/checkout.go::handleCheckout", Kind: "function", Path: "api/checkout.go",
			Name: "handleCheckout", Language: "go", IsExported: true},
		{ID: "internal/payment/p_test.go::TestCharge", Kind: "function", Path: "internal/payment/p_test.go",
			Name: "TestCharge", Language: "go"},
	}
	edges := []store.Edge{
		{Src: "api/checkout.go::handleCheckout", Dst: "internal/payment/p.go::Charge", Kind: "calls"},
		{Src: "internal/payment/p_test.go::TestCharge", Dst: "internal/payment/p.go::Charge", Kind: "tests"},
	}
	r := newStore(t, nodes, edges)

	in := ChangedSymbol{
		ID: "internal/payment/p.go::Charge", Kind: "method",
		IsExported: true, ChangeType: "modified",
	}
	got, err := enrichChangedSymbol(r, in, hubSet{}, 10)
	if err != nil {
		t.Fatal(err)
	}

	if got.Community != "internal/payment" {
		t.Errorf("community=%q", got.Community)
	}
	if got.Callers.Count != 1 || len(got.Callers.Sample) != 1 ||
		got.Callers.Sample[0] != "api/checkout.go::handleCheckout" {
		t.Errorf("callers=%+v", got.Callers)
	}
	if !got.Tests.HasTests || len(got.Tests.TestSymbols) != 1 {
		t.Errorf("tests=%+v", got.Tests)
	}
	wantFactors := []string{} // exported but tested, not in hub set, not interface
	sort.Strings(got.RiskFactors)
	if !reflect.DeepEqual(got.RiskFactors, wantFactors) && len(got.RiskFactors) != 0 {
		t.Errorf("risk factors=%v", got.RiskFactors)
	}
}
