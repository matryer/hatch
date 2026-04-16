// Package claude generates hatch output for Anthropic Claude Code.
//
// Claude Code reads project memory from CLAUDE.md, skills from
// .claude/skills/<name>/SKILL.md, user-invoked slash commands from
// .claude/commands/<name>.md, and sub-agents from .claude/agents/<name>.md.
// See https://code.claude.com/docs/en/ for the full surface.
package claude

import (
	"path/filepath"
	"strings"

	"github.com/grafana/hatch/pkg/render"
	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
)

const (
	name        = "claude"
	displayName = "Claude Code"
)

// Target is the Claude Code generator.
type Target struct{}

// New returns a Claude Code target. Callers pass this into a target.Set in
// main so there are no init-time registrations.
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

	// Rules → block inside CLAUDE.md.
	if body := target.RulesBlock(sc, name, displayName); body != "" {
		out = append(out, target.Artifact{
			Path:    target.ScopedPath(sc.Path, "CLAUDE.md"),
			Mode:    target.ModeBlock,
			Content: body,
		})
	}

	// Skills → .claude/skills/<name>/SKILL.md (+ sibling assets).
	for _, sk := range sc.Skills {
		if !sk.HasTarget(name) {
			continue
		}
		content, err := renderSkill(sk, displayName, name, hatchVersion, target.SourceFilePathFor(sc.Path, sk))
		if err != nil {
			return nil, err
		}
		skillDir := target.ScopedPath(sc.Path, ".claude", "skills", sk.Name)
		out = append(out, target.Artifact{
			Path:    filepath.Join(skillDir, "SKILL.md"),
			Mode:    target.ModeFile,
			Content: content,
		})
		// Sibling asset files copy through verbatim.
		assets, err := target.CopySkillAssets(sk, skillDir)
		if err != nil {
			return nil, err
		}
		out = append(out, assets...)
	}

	// Commands → .claude/commands/<name>.md.
	for _, c := range sc.Commands {
		if !c.HasTarget(name) {
			continue
		}
		content, err := renderSkill(c, displayName, name, hatchVersion, target.SourceFilePathFor(sc.Path, c))
		if err != nil {
			return nil, err
		}
		out = append(out, target.Artifact{
			Path:    target.ScopedPath(sc.Path, ".claude", "commands", c.Name+".md"),
			Mode:    target.ModeFile,
			Content: content,
		})
	}

	// Agents → .claude/agents/<name>.md.
	for _, a := range sc.Agents {
		if !a.HasTarget(name) {
			continue
		}
		content, err := renderSkill(a, displayName, name, hatchVersion, target.SourceFilePathFor(sc.Path, a))
		if err != nil {
			return nil, err
		}
		out = append(out, target.Artifact{
			Path:    target.ScopedPath(sc.Path, ".claude", "agents", a.Name+".md"),
			Mode:    target.ModeFile,
			Content: content,
		})
	}

	return out, nil
}

// renderSkill produces a SKILL.md for Claude Code. The same shape is also
// used for commands and agents — frontmatter with name + description plus
// any per-target passthrough fields the source supplied via a `claude:`
// block, plus a metadata block recording the hatch version and source
// path.
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
