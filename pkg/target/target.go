// Package target defines the Target interface and a Set type for managing
// the collection of targets a hatch invocation knows about. Each coding
// agent hatch knows how to generate files for lives in its own sub-package
// and is composed explicitly by the caller (typically cmd/hatch/main.go) —
// there are no init-time side effects.
package target

import (
	"fmt"
	"sort"
	"strings"

	"github.com/grafana/hatch/pkg/source"
)

// WriteMode describes how an artifact is written to disk.
type WriteMode string

const (
	// ModeFile writes the full file from scratch (overwriting existing).
	ModeFile WriteMode = "file"
	// ModeBlock injects the content into a hatch-managed block inside a file
	// that may have user-authored content around it.
	ModeBlock WriteMode = "block"
)

// Artifact is a single file (or block) hatch wants to write.
type Artifact struct {
	// Path is the output file path, relative to the project root.
	Path string
	// Mode is how to write it (full-file or block-injected).
	Mode WriteMode
	// Content is the bytes to write. For ModeFile this is the whole file;
	// for ModeBlock this is the block body (without marker lines).
	Content string
}

// Target is the per-agent code that turns a Source into artifacts.
type Target interface {
	// Name returns the short machine name, e.g. "claude".
	Name() string
	// DisplayName returns the human-readable name, e.g. "Claude Code".
	DisplayName() string
	// Generate returns the artifacts hatch should write for this target.
	Generate(s *source.Source) ([]Artifact, error)
}

// Set is an ordered collection of targets that supports name lookup and
// filtering. Construct one explicitly via NewSet; no globals.
type Set struct {
	targets []Target
	byName  map[string]Target
}

// NewSet builds a Set from the given targets. The input is sorted by Name
// so Set.All is deterministic regardless of caller order.
func NewSet(targets ...Target) *Set {
	sorted := make([]Target, len(targets))
	copy(sorted, targets)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name() < sorted[j].Name() })
	s := &Set{
		targets: sorted,
		byName:  make(map[string]Target, len(sorted)),
	}
	for _, t := range sorted {
		s.byName[t.Name()] = t
	}
	return s
}

// All returns every target in the set, sorted by name.
func (s *Set) All() []Target { return append([]Target(nil), s.targets...) }

// Names returns the target names in sorted order.
func (s *Set) Names() []string {
	out := make([]string, len(s.targets))
	for i, t := range s.targets {
		out[i] = t.Name()
	}
	return out
}

// Get returns the named target, or nil if unknown.
func (s *Set) Get(name string) Target { return s.byName[name] }

// Select returns a new Set containing only the named targets, preserving
// the caller-requested order. Unknown names return an error.
func (s *Set) Select(names []string) (*Set, error) {
	out := make([]Target, 0, len(names))
	for _, n := range names {
		t := s.Get(n)
		if t == nil {
			return nil, fmt.Errorf("unknown target %q (known: %s)", n, strings.Join(s.Names(), ", "))
		}
		out = append(out, t)
	}
	return &Set{targets: out, byName: copyLookup(out)}, nil
}

func copyLookup(ts []Target) map[string]Target {
	m := make(map[string]Target, len(ts))
	for _, t := range ts {
		m[t.Name()] = t
	}
	return m
}
