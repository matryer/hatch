// Package codex generates hatch output for the OpenAI Codex CLI.
//
// Codex reads project guidance from AGENTS.md (repo root) and skills from
// .agents/skills/<name>/SKILL.md (the agentskills.io standard path). Codex
// sub-agents live in ~/.codex/config.toml (TOML, not markdown) and slash
// commands are not a first-class primitive; hatch does not generate files
// for those. See https://developers.openai.com/codex/ for the full surface.
package codex

import (
	"path/filepath"
	"strings"

	"github.com/grafana/hatch/pkg/render"
	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
)

const (
	name        = "codex"
	displayName = "Codex"
)

// Target is the Codex CLI generator.
type Target struct{}

// New returns a Codex target.
func New() Target { return Target{} }

func (Target) Name() string        { return name }
func (Target) DisplayName() string { return displayName }

func (t Target) Generate(s *source.Source) ([]target.Artifact, error) {
	var out []target.Artifact
	for i := range s.Scopes {
		arts, err := t.generateScope(&s.Scopes[i], s.HatchVersion)
		if err != nil {
			return nil, err
		}
		out = append(out, arts...)
	}
	return out, nil
}

func (t Target) generateScope(sc *source.Scope, hatchVersion string) ([]target.Artifact, error) {
	var out []target.Artifact

	// Codex has no first-class slash-command or sub-agent primitive, so
	// hatch inlines commands and agents into AGENTS.md alongside the rules
	// block. The headings tell Codex how to interpret a user request that
	// matches one of those entries.
	sections := nonEmpty(
		target.RulesBlock(sc, name, displayName),
		target.CommandsBlock(sc, name, displayName),
		target.AgentsBlock(sc, name, displayName),
	)
	if len(sections) > 0 {
		out = append(out, target.Artifact{
			Path:    target.ScopedPath(sc.Path, "AGENTS.md"),
			Mode:    target.ModeBlock,
			Content: strings.Join(sections, "\n\n"),
		})
	}

	// Skills → .agents/skills/<name>/SKILL.md (agentskills.io standard path).
	for _, sk := range sc.Skills {
		if !sk.HasTarget(name) {
			continue
		}
		content, err := renderSkill(sk, displayName, name, hatchVersion, target.SourceFilePathFor(sc.Path, sk))
		if err != nil {
			return nil, err
		}
		dest := target.ScopedPath(sc.Path, ".agents", "skills", sk.Name)
		out = append(out, target.Artifact{
			Path:    filepath.Join(dest, "SKILL.md"),
			Mode:    target.ModeFile,
			Content: content,
		})
		assets, err := target.CopySkillAssets(sk, dest)
		if err != nil {
			return nil, err
		}
		out = append(out, assets...)
	}

	// Commands and agents: inlined into AGENTS.md above.

	return out, nil
}

// nonEmpty returns only the non-empty strings from its arguments.
func nonEmpty(sections ...string) []string {
	out := make([]string, 0, len(sections))
	for _, s := range sections {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func renderSkill(p source.Primitive, displayName, targetName, hatchVersion, sourcePath string) (string, error) {
	fields := []render.Field{
		{Key: "name", Value: p.Name},
		{Key: "description", Value: p.Description},
	}
	if over, ok := p.Overrides[name]; ok {
		fields = render.MergeOverride(fields, over)
	}
	fields = append(fields, target.MetadataField(hatchVersion, sourcePath))
	fm, err := render.Frontmatter(fields)
	if err != nil {
		return "", err
	}
	body := strings.TrimRight(render.Body(p.Body, displayName, targetName), "\n")
	if body == "" {
		return fm, nil
	}
	return fm + "\n" + body + "\n", nil
}
