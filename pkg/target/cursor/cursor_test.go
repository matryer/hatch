package cursor_test

import (
	"strings"
	"testing"

	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
	"github.com/grafana/hatch/pkg/target/cursor"
	"github.com/matryer/is"
)

func TestGenerate_RootRuleNoApplyTo_AlwaysAppliesMdc(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Rules: []source.Primitive{
			{Kind: source.KindRule, Name: "style", Body: "Be terse."},
		},
	}}}
	arts, err := cursor.New().Generate(s)
	is.NoErr(err)
	var rule *target.Artifact
	for i := range arts {
		if arts[i].Path == ".cursor/rules/style.mdc" {
			rule = &arts[i]
		}
	}
	is.True(rule != nil) // rule should land at .cursor/rules/<name>.mdc
	is.Equal(rule.Mode, target.ModeFile)
	is.True(strings.Contains(rule.Content, "alwaysApply: true"))
	is.True(strings.Contains(rule.Content, "Be terse."))
}

func TestGenerate_RootRuleWithApplyTo_GlobsFrontmatter(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Rules: []source.Primitive{
			{Kind: source.KindRule, Name: "go-rules", ApplyTo: "**/*.go", Body: "Go body."},
		},
	}}}
	arts, err := cursor.New().Generate(s)
	is.NoErr(err)
	var rule *target.Artifact
	for i := range arts {
		if arts[i].Path == ".cursor/rules/go-rules.mdc" {
			rule = &arts[i]
		}
	}
	is.True(rule != nil)
	is.True(strings.Contains(rule.Content, "globs:"))
	is.True(strings.Contains(rule.Content, "**/*.go"))
	is.True(strings.Contains(rule.Content, "alwaysApply: false"))
}

func TestGenerate_SkillBecomesInlineMdc(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Skills: []source.Primitive{{
			Kind: source.KindSkill, Name: "review", Description: "Review a PR.", Body: "skill body",
		}},
	}}}
	arts, err := cursor.New().Generate(s)
	is.NoErr(err)
	var sk *target.Artifact
	for i := range arts {
		if arts[i].Path == ".cursor/rules/skill-review.mdc" {
			sk = &arts[i]
		}
	}
	is.True(sk != nil)
	is.True(strings.Contains(sk.Content, "alwaysApply: true"))
	is.True(strings.Contains(sk.Content, "description: Review a PR."))
	is.True(strings.Contains(sk.Content, "skill body"))
}

func TestGenerate_CommandAndAgentBecomeInlineMdc(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Commands: []source.Primitive{{
			Kind: source.KindCommand, Name: "commit", Description: "d", Body: "cmd body",
		}},
		Agents: []source.Primitive{{
			Kind: source.KindAgent, Name: "guard", Description: "d", Body: "agent body",
		}},
	}}}
	arts, err := cursor.New().Generate(s)
	is.NoErr(err)
	paths := map[string]string{}
	for _, a := range arts {
		paths[a.Path] = a.Content
	}
	is.True(strings.Contains(paths[".cursor/rules/command-commit.mdc"], "cmd body"))
	is.True(strings.Contains(paths[".cursor/rules/agent-guard.mdc"], "agent body"))
}

func TestGenerate_NamespacedCommandFilenameFlattened(t *testing.T) {
	// Cursor's command- prefix convention can't carry a namespace, so a
	// source command named "opsx/apply" renders as
	// .cursor/rules/command-opsx-apply.mdc (slash → dash).
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Commands: []source.Primitive{{
			Kind: source.KindCommand, Name: "opsx/apply", Description: "d", Body: "apply body",
		}},
	}}}
	arts, err := cursor.New().Generate(s)
	is.NoErr(err)
	paths := map[string]bool{}
	for _, a := range arts {
		paths[a.Path] = true
	}
	is.True(paths[".cursor/rules/command-opsx-apply.mdc"])
	is.True(!paths[".cursor/rules/command-opsx/apply.mdc"])
}

func TestGenerate_MdcFilesHaveMetadata(t *testing.T) {
	is := is.New(t)
	s := &source.Source{
		HatchVersion: "v0.9.9-test",
		Scopes: []source.Scope{{
			Rules: []source.Primitive{{
				Kind: source.KindRule, Name: "style", Body: "rule body", SourcePath: "x",
			}},
			Skills: []source.Primitive{{
				Kind: source.KindSkill, Name: "review-pr", Description: "d", Body: "b", SourcePath: "x",
			}},
		}},
	}
	arts, err := cursor.New().Generate(s)
	is.NoErr(err)
	byPath := map[string]string{}
	for _, a := range arts {
		byPath[a.Path] = a.Content
	}
	rule := byPath[".cursor/rules/style.mdc"]
	is.True(strings.Contains(rule, "metadata:"))
	is.True(strings.Contains(rule, "generated: hatch@v0.9.9-test"))
	is.True(strings.Contains(rule, "source: .hatch/_rules/style.md"))

	skill := byPath[".cursor/rules/skill-review-pr.mdc"]
	is.True(strings.Contains(skill, "metadata:"))
	is.True(strings.Contains(skill, "source: .hatch/_skills/review-pr/SKILL.md"))
}

func TestGenerate_ScopedRuleSluggedAndGlobbed(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: ""},
		{
			Path: "backend",
			Rules: []source.Primitive{
				{Kind: source.KindRule, Name: "style", Body: "BACKEND RULE"},
			},
		},
	}}
	arts, err := cursor.New().Generate(s)
	is.NoErr(err)
	var rule *target.Artifact
	for i := range arts {
		if arts[i].Path == ".cursor/rules/backend-style.mdc" {
			rule = &arts[i]
		}
	}
	is.True(rule != nil)
	is.True(strings.Contains(rule.Content, "backend/**"))
	is.True(strings.Contains(rule.Content, "alwaysApply: false"))
	is.True(strings.Contains(rule.Content, "BACKEND RULE"))
	// And no nested .cursor/ tree was emitted.
	for _, a := range arts {
		if strings.HasPrefix(a.Path, "backend/") {
			t.Fatalf("scoped Cursor output must stay at root, got %q", a.Path)
		}
	}
}

func TestGenerate_ScopedRule_AbsoluteApplyToErrors(t *testing.T) {
	is := is.New(t)
	for _, glob := range []string{"/etc/foo", "**/foo.go"} {
		s := &source.Source{Scopes: []source.Scope{
			{Path: ""},
			{
				Path: "backend",
				Rules: []source.Primitive{{
					Kind: source.KindRule, Name: "bad", ApplyTo: glob, Body: "x",
				}},
			},
		}}
		_, err := cursor.New().Generate(s)
		is.True(err != nil)
		is.True(strings.Contains(err.Error(), "bad"))
	}
}

func TestGenerate_ScopedSkill_LabelInBody(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: ""},
		{
			Path: "backend",
			Skills: []source.Primitive{{
				Kind: source.KindSkill, Name: "review", Description: "d", Body: "scoped skill body",
			}},
		},
	}}
	arts, err := cursor.New().Generate(s)
	is.NoErr(err)
	var sk *target.Artifact
	for i := range arts {
		if arts[i].Path == ".cursor/rules/skill-backend-review.mdc" {
			sk = &arts[i]
		}
	}
	is.True(sk != nil)
	is.True(strings.Contains(sk.Content, "skill: backend/review"))
	is.True(strings.Contains(sk.Content, "scoped skill body"))
}

func TestGenerate_RespectsTargetsFilter(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Rules: []source.Primitive{{
			Kind: source.KindRule, Name: "only-claude", Body: "only-claude", Targets: []string{"claude"},
		}},
	}}}
	arts, err := cursor.New().Generate(s)
	is.NoErr(err)
	is.Equal(len(arts), 0)
}
