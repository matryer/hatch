// Package copilot generates hatch output for GitHub Copilot.
//
// Copilot reads project instructions from .github/copilot-instructions.md,
// path-scoped instructions from .github/instructions/<name>.instructions.md
// (with an `applyTo` glob), prompt files from .github/prompts/<name>.prompt.md,
// and custom agents from .github/agents/<name>.agent.md. Copilot has no
// documented skill primitive: hatch inlines skill bodies into the hatch block
// inside copilot-instructions.md so that skill content still reaches Copilot.
//
// Copilot only reads .github/* from the repository root — it does not pick
// up nested .github/ directories the way Claude Code and Codex pick up
// nested CLAUDE.md / AGENTS.md. To keep nested-scope content reachable,
// hatch routes scoped Copilot output through Copilot's native scoping
// mechanism (`.github/instructions/<name>.instructions.md` with an applyTo
// glob) and rewrites filenames with a scope-derived slug to avoid
// collisions across paths.
//
// See https://docs.github.com/copilot/ for the full surface.
package copilot

import (
	"path/filepath"
	"strings"

	"github.com/grafana/hatch/pkg/render"
	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
)

const (
	name        = "copilot"
	displayName = "GitHub Copilot"
)

// Target is the Copilot generator.
type Target struct{}

// New returns a Copilot target.
func New() Target { return Target{} }

func (Target) Name() string        { return name }
func (Target) DisplayName() string { return displayName }

func (t Target) Generate(s *source.Source) ([]target.Artifact, error) {
	var out []target.Artifact

	// Single root .github/copilot-instructions.md block, combining:
	//   - root scope's rules without applyTo
	//   - skills from every scope, with scope-labelled headings for nested
	var buf strings.Builder
	if root := s.Root(); root != nil {
		rulesBody := unscopedRulesBlock(root, name, displayName)
		if rulesBody != "" {
			buf.WriteString(rulesBody)
		}
	}
	skillsBody := scopedSkillsItems(s, name, displayName)
	if skillsBody != "" {
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString("## Skills\n\nThe following capabilities describe actions the user may ask the model to perform.\n")
		buf.WriteString(skillsBody)
	}
	if buf.Len() > 0 {
		out = append(out, target.Artifact{
			Path:    filepath.Join(".github", "copilot-instructions.md"),
			Mode:    target.ModeBlock,
			Content: buf.String(),
		})
	}

	// Per-scope file-owned outputs. Copilot reads all of these from the
	// repo root only — the scope shows up in the *filename* (slug prefix)
	// and, for rules, in the *applyTo glob* — never in the path.
	for i := range s.Scopes {
		sc := &s.Scopes[i]
		slug := scopeSlug(sc.Path)
		prefix := ""
		if slug != "" {
			prefix = slug + "-"
		}

		// Rules → .github/instructions/<prefix><name>.instructions.md
		//
		// Root scope keeps today's behaviour: only rules WITH applyTo
		// become instructions files; rules without applyTo are inlined
		// into the block above. Nested scopes emit ALL their rules as
		// instructions files (with composeApplyTo doing the work to
		// derive the right glob), since nested rules can't go in the
		// root block without losing their scope.
		for _, r := range sc.Rules {
			if !r.HasTarget(name) {
				continue
			}
			if sc.Path == "" && r.ApplyTo == "" {
				continue
			}
			applyTo, err := composeApplyTo(sc.Path, r.ApplyTo, r.Name)
			if err != nil {
				return nil, err
			}
			content, err := renderScopedRule(r, applyTo, displayName, name, s.HatchVersion, target.SourceFilePathFor(sc.Path, r))
			if err != nil {
				return nil, err
			}
			out = append(out, target.Artifact{
				Path:    filepath.Join(".github", "instructions", prefix+r.Name+".instructions.md"),
				Mode:    target.ModeFile,
				Content: content,
			})
		}

		// Commands → .github/prompts/<prefix><name>.prompt.md.
		for _, c := range sc.Commands {
			if !c.HasTarget(name) {
				continue
			}
			content, err := renderSlashPrimitive(c, displayName, name, s.HatchVersion, target.SourceFilePathFor(sc.Path, c))
			if err != nil {
				return nil, err
			}
			out = append(out, target.Artifact{
				Path:    filepath.Join(".github", "prompts", prefix+target.FlatName(c.Name)+".prompt.md"),
				Mode:    target.ModeFile,
				Content: content,
			})
		}

		// Agents → .github/agents/<prefix><name>.agent.md.
		for _, a := range sc.Agents {
			if !a.HasTarget(name) {
				continue
			}
			content, err := renderSlashPrimitive(a, displayName, name, s.HatchVersion, target.SourceFilePathFor(sc.Path, a))
			if err != nil {
				return nil, err
			}
			out = append(out, target.Artifact{
				Path:    filepath.Join(".github", "agents", prefix+a.Name+".agent.md"),
				Mode:    target.ModeFile,
				Content: content,
			})
		}
	}

	return out, nil
}

// unscopedRulesBlock returns rule bodies concatenated as markdown for rules
// that do NOT have an `applyTo` glob. Path-scoped rules are generated as
// separate `.github/instructions/` files instead of being inlined.
func unscopedRulesBlock(sc *source.Scope, targetName, displayName string) string {
	var buf strings.Builder
	first := true
	for _, r := range sc.Rules {
		if !r.HasTarget(targetName) || r.ApplyTo != "" {
			continue
		}
		body := strings.TrimSpace(render.Body(r.Body, displayName, targetName))
		if body == "" {
			continue
		}
		if !first {
			buf.WriteString("\n\n")
		}
		first = false
		buf.WriteString(body)
	}
	return buf.String()
}

// scopedSkillsItems renders the skill section items for every scope into
// a single combined block, suitable for placement under a top-level
// "## Skills" heading. Root-scope skills get a "### Skill: <name>"
// heading; nested-scope skills get "### Skill: <scope>/<name>" so the
// reader can tell which scope each skill came from. Returns an empty
// string if no skills apply to the target across any scope.
func scopedSkillsItems(s *source.Source, targetName, displayName string) string {
	type entry struct {
		label string
		prim  source.Primitive
	}
	var items []entry
	for i := range s.Scopes {
		sc := &s.Scopes[i]
		for _, sk := range sc.Skills {
			if !sk.HasTarget(targetName) {
				continue
			}
			label := sk.Name
			if sc.Path != "" {
				label = sc.Path + "/" + sk.Name
			}
			items = append(items, entry{label: label, prim: sk})
		}
	}
	if len(items) == 0 {
		return ""
	}
	var buf strings.Builder
	for _, it := range items {
		body := strings.TrimSpace(render.Body(it.prim.Body, displayName, targetName))
		buf.WriteString("\n### Skill: ")
		buf.WriteString(it.label)
		buf.WriteString("\n\n")
		if it.prim.Description != "" {
			buf.WriteString("_")
			buf.WriteString(it.prim.Description)
			buf.WriteString("_\n\n")
		}
		buf.WriteString(body)
		buf.WriteString("\n")
	}
	return strings.TrimRight(buf.String(), "\n")
}

func renderScopedRule(p source.Primitive, applyTo, displayName, targetName, hatchVersion, sourcePath string) (string, error) {
	fields := []render.Field{
		{Key: "applyTo", Value: applyTo},
	}
	if p.Description != "" {
		fields = append(fields, render.Field{Key: "description", Value: p.Description})
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

func renderSlashPrimitive(p source.Primitive, displayName, targetName, hatchVersion, sourcePath string) (string, error) {
	fields := []render.Field{
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
