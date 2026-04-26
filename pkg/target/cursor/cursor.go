// Package cursor generates hatch output for Cursor (https://cursor.sh).
//
// Cursor reads project rules from `.cursor/rules/*.mdc` files. Each .mdc
// file has a YAML frontmatter header with three optional keys:
//
//   - description: human-readable summary
//   - globs: list of glob patterns scoping the rule to matching files
//   - alwaysApply: bool — load the rule for every request when true
//
// Cursor has no native skill, slash-command, or sub-agent primitive, so
// hatch represents the other three primitives as additional .mdc rule
// files with a kind-prefixed filename and `alwaysApply: true`. The
// model still gets the content; users authoring for Cursor see the
// boundary between hatch-managed inline rules and their own rules at a
// glance via the kind prefix in the filename.
//
// Cursor only reads `.cursor/rules/` from the repository root — it does
// not pick up nested `.cursor/` directories. To keep nested-scope content
// reachable, hatch routes scoped Cursor output through Cursor's native
// scoping mechanism (frontmatter `globs`) and rewrites filenames with a
// scope-derived slug to avoid collisions across paths. This mirrors how
// hatch handles GitHub Copilot.
package cursor

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/grafana/hatch/pkg/render"
	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
)

const (
	name        = "cursor"
	displayName = "Cursor"
)

// Target is the Cursor generator.
type Target struct{}

// New returns a Cursor target.
func New() Target { return Target{} }

func (Target) Name() string        { return name }
func (Target) DisplayName() string { return displayName }

func (t Target) Generate(s *source.Source) ([]target.Artifact, error) {
	var out []target.Artifact

	for i := range s.Scopes {
		sc := &s.Scopes[i]
		slug := scopeSlug(sc.Path)
		prefix := ""
		if slug != "" {
			prefix = slug + "-"
		}

		// Rules → .cursor/rules/<prefix><name>.mdc
		for _, r := range sc.Rules {
			if !r.HasTarget(name) {
				continue
			}
			globs, err := composeGlobs(sc.Path, r.ApplyTo, r.Name)
			if err != nil {
				return nil, err
			}
			alwaysApply := len(globs) == 0
			content, err := renderRuleMdc(r, globs, alwaysApply, displayName, name, s.HatchVersion, target.SourceFilePathFor(sc.Path, r))
			if err != nil {
				return nil, err
			}
			out = append(out, target.Artifact{
				Path:    filepath.Join(".cursor", "rules", prefix+r.Name+".mdc"),
				Mode:    target.ModeFile,
				Content: content,
			})
		}

		// Skills → .cursor/rules/skill-<prefix><name>.mdc (alwaysApply,
		// scope-labelled in the body so the model can tell which scope a
		// skill came from when hatch is used in a monorepo).
		for _, sk := range sc.Skills {
			if !sk.HasTarget(name) {
				continue
			}
			body := scopeLabelledBody(sc.Path, sk, displayName)
			content, err := renderInlineMdc(sk.Description, body, s.HatchVersion, target.SourceFilePathFor(sc.Path, sk))
			if err != nil {
				return nil, err
			}
			out = append(out, target.Artifact{
				Path:    filepath.Join(".cursor", "rules", "skill-"+prefix+sk.Name+".mdc"),
				Mode:    target.ModeFile,
				Content: content,
			})
		}

		// Commands → .cursor/rules/command-<prefix><name>.mdc
		for _, c := range sc.Commands {
			if !c.HasTarget(name) {
				continue
			}
			body := scopeLabelledBody(sc.Path, c, displayName)
			content, err := renderInlineMdc(c.Description, body, s.HatchVersion, target.SourceFilePathFor(sc.Path, c))
			if err != nil {
				return nil, err
			}
			out = append(out, target.Artifact{
				Path:    filepath.Join(".cursor", "rules", "command-"+prefix+target.FlatName(c.Name)+".mdc"),
				Mode:    target.ModeFile,
				Content: content,
			})
		}

		// Agents → .cursor/rules/agent-<prefix><name>.mdc
		for _, a := range sc.Agents {
			if !a.HasTarget(name) {
				continue
			}
			body := scopeLabelledBody(sc.Path, a, displayName)
			content, err := renderInlineMdc(a.Description, body, s.HatchVersion, target.SourceFilePathFor(sc.Path, a))
			if err != nil {
				return nil, err
			}
			out = append(out, target.Artifact{
				Path:    filepath.Join(".cursor", "rules", "agent-"+prefix+a.Name+".mdc"),
				Mode:    target.ModeFile,
				Content: content,
			})
		}
	}

	return out, nil
}

// composeGlobs combines a scope path with the user-supplied applyTo
// glob into Cursor's `globs` frontmatter list. Cursor's native scoping
// is via globs, so this is the analogue of the Copilot composeApplyTo
// helper: same rules (no-glob root → no globs; scoped no-glob →
// "<scope>/**"; scoped + relative glob → prepended; scoped + absolute
// or **/-prefixed glob → error).
func composeGlobs(scopePath, userGlob, ruleName string) ([]string, error) {
	if scopePath == "" {
		if userGlob == "" {
			return nil, nil
		}
		return []string{userGlob}, nil
	}
	if userGlob == "" {
		return []string{scopePath + "/**"}, nil
	}
	if strings.HasPrefix(userGlob, "/") || strings.HasPrefix(userGlob, "**/") {
		return nil, fmt.Errorf("rule %q at path %q: explicit applyTo %q is absolute or unanchored; remove applyTo to inherit the path glob, or move the rule to the root .hatch/_rules/", ruleName, scopePath, userGlob)
	}
	return []string{scopePath + "/" + userGlob}, nil
}

// scopeSlug converts a forward-slash scope path into a flat slug
// suitable for use in a filename: services/api -> services-api.
func scopeSlug(scopePath string) string {
	if scopePath == "" {
		return ""
	}
	return strings.ReplaceAll(scopePath, "/", "-")
}

// renderRuleMdc renders a Cursor rule .mdc file for the given primitive,
// with the supplied globs/alwaysApply frontmatter values. Per-target
// `cursor:` passthrough fields from the source are merged into the
// frontmatter alongside.
func renderRuleMdc(p source.Primitive, globs []string, alwaysApply bool, displayName, targetName, hatchVersion, sourcePath string) (string, error) {
	fields := []render.Field{}
	if p.Description != "" {
		fields = append(fields, render.Field{Key: "description", Value: p.Description})
	}
	if len(globs) > 0 {
		fields = append(fields, render.Field{Key: "globs", Value: globs})
	}
	fields = append(fields, render.Field{Key: "alwaysApply", Value: alwaysApply})
	if over, ok := p.Overrides[name]; ok {
		fields = render.MergeOverride(fields, over)
	}
	fields = target.MergeField(fields, target.MetadataField(hatchVersion, sourcePath))
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

// renderInlineMdc renders an "inlined" .mdc rule file used for skills,
// commands, and agents (which have no native Cursor primitive). These
// always have alwaysApply: true so the body is visible in every
// session, mirroring how Copilot inlines the same primitives.
func renderInlineMdc(description, body, hatchVersion, sourcePath string) (string, error) {
	fields := []render.Field{}
	if description != "" {
		fields = append(fields, render.Field{Key: "description", Value: description})
	}
	fields = append(fields, render.Field{Key: "alwaysApply", Value: true})
	fields = target.MergeField(fields, target.MetadataField(hatchVersion, sourcePath))
	fm, err := render.Frontmatter(fields)
	if err != nil {
		return "", err
	}
	body = strings.TrimRight(body, "\n")
	if body == "" {
		return fm, nil
	}
	return fm + "\n" + body + "\n", nil
}

// scopeLabelledBody returns the primitive's body, prefixed with a
// scope-aware H2 heading when the primitive lives in a non-root scope.
// This makes the source scope visible in the generated .mdc body so a
// reader looking at .cursor/rules/skill-backend-review.mdc immediately
// sees that the content came from the backend scope.
func scopeLabelledBody(scopePath string, p source.Primitive, displayName string) string {
	body := strings.TrimRight(render.Body(p.Body, displayName, name), "\n")
	if scopePath == "" {
		return body
	}
	var buf strings.Builder
	buf.WriteString("## ")
	buf.WriteString(string(p.Kind))
	buf.WriteString(": ")
	buf.WriteString(scopePath)
	buf.WriteString("/")
	buf.WriteString(target.FlatName(p.Name))
	buf.WriteString("\n\n")
	buf.WriteString(body)
	return buf.String()
}
