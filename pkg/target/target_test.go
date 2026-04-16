package target_test

import (
	"testing"

	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
	"github.com/matryer/is"
)

// stubTarget is a minimal Target used only by set tests.
type stubTarget struct {
	name    string
	display string
}

func (s stubTarget) Name() string                                       { return s.name }
func (s stubTarget) DisplayName() string                                { return s.display }
func (s stubTarget) Generate(*source.Source) ([]target.Artifact, error) { return nil, nil }

func TestSet_NamesSorted(t *testing.T) {
	is := is.New(t)
	s := target.NewSet(
		stubTarget{name: "zeta", display: "Zeta"},
		stubTarget{name: "alpha", display: "Alpha"},
	)
	is.Equal(s.Names(), []string{"alpha", "zeta"})
}

func TestSet_Get(t *testing.T) {
	is := is.New(t)
	s := target.NewSet(stubTarget{name: "alpha"}, stubTarget{name: "beta"})
	got := s.Get("beta")
	is.True(got != nil)
	is.Equal(got.Name(), "beta")
	is.Equal(s.Get("missing"), target.Target(nil))
}

func TestSet_SelectKnown(t *testing.T) {
	is := is.New(t)
	s := target.NewSet(
		stubTarget{name: "a"},
		stubTarget{name: "b"},
		stubTarget{name: "c"},
	)
	sub, err := s.Select([]string{"c", "a"})
	is.NoErr(err)
	// Select preserves caller order.
	names := make([]string, len(sub.All()))
	for i, t := range sub.All() {
		names[i] = t.Name()
	}
	is.Equal(names, []string{"c", "a"})
}

func TestSet_SelectUnknownErrors(t *testing.T) {
	is := is.New(t)
	s := target.NewSet(stubTarget{name: "a"})
	_, err := s.Select([]string{"missing"})
	is.True(err != nil) // unknown target should error
}
