package zed_test

import (
	"strings"
	"testing"

	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
	"github.com/grafana/hatch/pkg/target/zed"
	"github.com/matryer/is"
)

func TestGenerate_RootRuleAsBlockInRules(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Rules: []source.Primitive{
			{Kind: source.KindRule, Name: "style", Body: "be terse"},
		},
	}}}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)
	blk := findArtifact(arts, ".rules")
	is.True(blk != nil)
	is.Equal(blk.Mode, target.ModeBlock)
	is.True(strings.Contains(blk.Content, "be terse"))
}

func TestGenerate_RuleWithApplyToInlinesHeading(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Rules: []source.Primitive{
			{Kind: source.KindRule, Name: "go-tests", ApplyTo: "**/*_test.go", Body: "use is.New(t)"},
		},
	}}}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)
	blk := findArtifact(arts, ".rules")
	is.True(blk != nil)
	is.True(strings.Contains(blk.Content, "**/*_test.go"))
	is.True(strings.Contains(blk.Content, "use is.New(t)"))
}

func TestGenerate_CommandInlinedUnderCommandsHeading(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Commands: []source.Primitive{
			{Kind: source.KindCommand, Name: "review-pr", Description: "review a PR", Body: "do the review"},
		},
	}}}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)
	blk := findArtifact(arts, ".rules")
	is.True(blk != nil)
	is.True(strings.Contains(blk.Content, "## Commands"))
	is.True(strings.Contains(blk.Content, "review-pr"))
	is.True(strings.Contains(blk.Content, "do the review"))
}

func TestGenerate_AgentInlinedUnderSubAgentsHeading(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Agents: []source.Primitive{
			{Kind: source.KindAgent, Name: "researcher", Description: "research", Body: "do research"},
		},
	}}}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)
	blk := findArtifact(arts, ".rules")
	is.True(blk != nil)
	is.True(strings.Contains(blk.Content, "## Sub-agents"))
	is.True(strings.Contains(blk.Content, "researcher"))
	is.True(strings.Contains(blk.Content, "do research"))
}

func TestGenerate_SkillInlinedUnderSkillsHeading(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Skills: []source.Primitive{
			{Kind: source.KindSkill, Name: "review-pr", Description: "review a PR", Body: "skill body"},
		},
	}}}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)
	blk := findArtifact(arts, ".rules")
	is.True(blk != nil)
	is.True(strings.Contains(blk.Content, "## Skills"))
	is.True(strings.Contains(blk.Content, "review-pr"))
	is.True(strings.Contains(blk.Content, "skill body"))

	// No separate skill files written — .rules is the only artifact.
	for _, a := range arts {
		is.True(!strings.Contains(a.Path, "skills/"))
	}
}

func TestGenerate_NestedScopeFlattenedIntoRootRules(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: "", Rules: []source.Primitive{{Kind: source.KindRule, Name: "global", Body: "ROOT RULE"}}},
		{Path: "backend", Rules: []source.Primitive{{Kind: source.KindRule, Name: "be-style", Body: "BACKEND RULE"}}},
	}}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)

	// Only one .rules artifact, at project root.
	count := 0
	for _, a := range arts {
		if strings.HasSuffix(a.Path, ".rules") {
			count++
		}
	}
	is.Equal(count, 1)

	blk := findArtifact(arts, ".rules")
	is.True(blk != nil)
	is.True(strings.Contains(blk.Content, "ROOT RULE"))
	is.True(strings.Contains(blk.Content, "BACKEND RULE"))
	is.True(strings.Contains(blk.Content, "# Scope: backend/")) // exact label format

	// Root content precedes the scope header; scope header precedes backend rule.
	rootIdx := strings.Index(blk.Content, "ROOT RULE")
	scopeIdx := strings.Index(blk.Content, "# Scope: backend/")
	beIdx := strings.Index(blk.Content, "BACKEND RULE")
	is.True(rootIdx >= 0 && scopeIdx > rootIdx && beIdx > scopeIdx)
}

func TestGenerate_MultipleNestedScopesEachLabeled(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: ""},
		{Path: "backend", Rules: []source.Primitive{{Kind: source.KindRule, Name: "be", Body: "BE BODY"}}},
		{Path: "frontend", Commands: []source.Primitive{{Kind: source.KindCommand, Name: "fe-cmd", Description: "d", Body: "FE CMD BODY"}}},
	}}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)
	blk := findArtifact(arts, ".rules")
	is.True(blk != nil)
	is.True(strings.Contains(blk.Content, "# Scope: backend/"))
	is.True(strings.Contains(blk.Content, "# Scope: frontend/"))
	is.True(strings.Contains(blk.Content, "BE BODY"))
	is.True(strings.Contains(blk.Content, "FE CMD BODY"))
	// Backend (lex first) appears before frontend.
	is.True(strings.Index(blk.Content, "# Scope: backend/") < strings.Index(blk.Content, "# Scope: frontend/"))
}

func TestGenerate_TargetFilteringExcludesUnselectedPrimitives(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Rules: []source.Primitive{
			{Kind: source.KindRule, Name: "claude-only", Body: "CLAUDE ONLY", Targets: []string{"claude"}},
			{Kind: source.KindRule, Name: "all", Body: "EVERYWHERE"},
		},
	}}}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)
	blk := findArtifact(arts, ".rules")
	is.True(blk != nil)
	is.True(strings.Contains(blk.Content, "EVERYWHERE"))
	is.True(!strings.Contains(blk.Content, "CLAUDE ONLY"))
}

func TestGenerate_EmptySourceProducesNoArtifacts(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{}}}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)
	is.Equal(len(arts), 0)
}

func TestNameAndDisplayName(t *testing.T) {
	is := is.New(t)
	t1 := zed.New()
	is.Equal(t1.Name(), "zed")
	is.Equal(t1.DisplayName(), "Zed")
}

func findArtifact(arts []target.Artifact, path string) *target.Artifact {
	for i := range arts {
		if arts[i].Path == path {
			return &arts[i]
		}
	}
	return nil
}
