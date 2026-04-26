// Package zed generates hatch output for the Zed editor.
//
// Zed reads project guidance from a single rules file at the repo root,
// picking the first match in a priority list (`.rules`, `.cursorrules`,
// …, `AGENTS.md`, `CLAUDE.md`, …). Hatch writes to `.rules` — Zed's
// highest-priority filename — to avoid colliding with codex/opencode,
// which already own AGENTS.md.
//
// Zed has no project-level slash commands, sub-agents, or skills
// (native skill support is in flight: zed-industries/zed#49057).
// Hatch inlines all four primitives — rules, skills, commands, and
// sub-agents — into the rules file as markdown sections so Zed sees
// the maximum coverage of the source spec. Sibling skill assets are
// not copied because `.rules` is the only file Zed reads. Nested
// scopes flatten into the same root file under `# Scope: <path>/`
// headings.
//
// See https://zed.dev/docs/ai/rules for the rules-file contract.
package zed

import (
	"strings"

	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
)

const (
	name        = "zed"
	displayName = "Zed"
)

// Target is the Zed generator.
type Target struct{}

// New returns a Zed target.
func New() Target { return Target{} }

func (Target) Name() string        { return name }
func (Target) DisplayName() string { return displayName }

func (t Target) Generate(s *source.Source) ([]target.Artifact, error) {
	var sections []string
	for i := range s.Scopes {
		sc := &s.Scopes[i]
		block := scopeBlock(sc)
		if block == "" {
			continue
		}
		if sc.Path != "" {
			block = "# Scope: " + sc.Path + "/\n\n" + block
		}
		sections = append(sections, block)
	}
	if len(sections) == 0 {
		return nil, nil
	}
	return []target.Artifact{{
		Path:    ".rules",
		Mode:    target.ModeBlock,
		Content: strings.Join(sections, "\n\n"),
	}}, nil
}

// scopeBlock renders one scope's content (rules + skills + commands +
// agents) as the markdown body that goes inside the hatch-managed
// block.
func scopeBlock(sc *source.Scope) string {
	parts := nonEmpty(
		target.RulesBlock(sc, name, displayName),
		target.SkillsBlock(sc, name, displayName),
		target.CommandsBlock(sc, name, displayName),
		target.AgentsBlock(sc, name, displayName),
	)
	return strings.Join(parts, "\n\n")
}

func nonEmpty(sections ...string) []string {
	out := make([]string, 0, len(sections))
	for _, s := range sections {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}
