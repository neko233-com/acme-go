package dnsprovider

import (
	"path/filepath"
	"testing"

	"github.com/neko233-com/acme233/internal/testspec"
)

type providerAliasSpec struct {
	Canonical string   `json:"canonical"`
	Inputs    []string `json:"inputs"`
}

func TestResolveSupportsVendorAliases(t *testing.T) {
	var specs []providerAliasSpec
	if err := testspec.LoadJSON(filepath.Join("specs", "provider_aliases.json"), &specs); err != nil {
		t.Fatalf("load provider specs: %v", err)
	}

	for _, spec := range specs {
		for _, input := range spec.Inputs {
			strategy, err := Resolve(input)
			if err != nil {
				t.Fatalf("resolve %q: %v", input, err)
			}
			if strategy.ID() != spec.Canonical {
				t.Fatalf("resolve %q: got %q want %q", input, strategy.ID(), spec.Canonical)
			}
		}
	}
}
