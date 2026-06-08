// Package zed generates hatch output for the Zed editor.
//
// Zed reads project guidance from a single rules file at the repo root,
// picking the first match in a priority list (`.rules`, `.cursorrules`,
// …, `AGENTS.md`, `CLAUDE.md`, …). Hatch writes to `.rules` — Zed's
// highest-priority filename — to avoid colliding with codex/opencode,
// which already own AGENTS.md.
//
// Zed has native skill support: skills are written as individual
// `.agents/skills/<name>/SKILL.md` file artifacts, with sibling assets
// copied verbatim. Rules, commands, and sub-agents — which have no native
// Zed primitive — are inlined into `.rules` as markdown sections. Nested
// scopes flatten into the same root file under `# Scope: <path>/` headings.
//
// See https://zed.dev/docs/ai/rules for the rules-file contract.
// See https://zed.dev/docs/ai/skills for the skills contract.
package zed

import (
	"path/filepath"
	"strings"

	"github.com/grafana/hatch/pkg/render"
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
	var out []target.Artifact

	// Native skill files — written per-skill, per-scope.
	for i := range s.Scopes {
		arts, err := t.generateSkills(&s.Scopes[i], s.HatchVersion)
		if err != nil {
			return nil, err
		}
		out = append(out, arts...)
	}

	// Inline rules, commands, and agents into .rules.
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
	if len(sections) > 0 {
		out = append(out, target.Artifact{
			Path:    ".rules",
			Mode:    target.ModeBlock,
			Content: strings.Join(sections, "\n\n"),
		})
	}

	return out, nil
}

// generateSkills writes native .agents/skills/<name>/SKILL.md artifacts for
// each skill in the scope, plus any sibling asset files.
func (t Target) generateSkills(sc *source.Scope, hatchVersion string) ([]target.Artifact, error) {
	var out []target.Artifact
	for _, sk := range sc.Skills {
		if !sk.HasTarget(name) {
			continue
		}
		content, err := renderSkill(sk, hatchVersion, target.SourceFilePathFor(sc.Path, sk))
		if err != nil {
			return nil, err
		}
		skillDir := target.ScopedPath(sc.Path, ".agents", "skills", sk.Name)
		out = append(out, target.Artifact{
			Path:    filepath.Join(skillDir, "SKILL.md"),
			Mode:    target.ModeFile,
			Content: content,
		})
		assets, err := target.CopySkillAssets(sk, skillDir)
		if err != nil {
			return nil, err
		}
		out = append(out, assets...)
	}
	return out, nil
}

// scopeBlock renders one scope's content (rules + commands + agents) as the
// markdown body that goes inside the hatch-managed block. Skills are emitted
// as native files and are not inlined here.
func scopeBlock(sc *source.Scope) string {
	parts := nonEmpty(
		target.RulesBlock(sc, name, displayName),
		target.CommandsBlock(sc, name, displayName),
		target.AgentsBlock(sc, name, displayName),
	)
	return strings.Join(parts, "\n\n")
}

// renderSkill produces a SKILL.md for Zed. Frontmatter contains name and
// description; any per-target `zed:` override block is merged on top.
func renderSkill(p source.Primitive, hatchVersion, sourcePath string) (string, error) {
	fields := []render.Field{
		{Key: "name", Value: p.Name},
		{Key: "description", Value: p.Description},
	}
	if over, ok := p.Overrides[name]; ok {
		fields = render.MergeOverride(fields, over)
	}
	fields = target.MergeField(fields, target.MetadataField(hatchVersion, sourcePath))
	fm, err := render.Frontmatter(fields)
	if err != nil {
		return "", err
	}
	body := strings.TrimRight(render.Body(p.Body, displayName, name), "\n")
	if body == "" {
		return fm, nil
	}
	return fm + "\n" + body + "\n", nil
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
