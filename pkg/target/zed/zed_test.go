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

func TestGenerate_SkillWrittenAsNativeFile(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Skills: []source.Primitive{
			{Kind: source.KindSkill, Name: "review-pr", Description: "review a PR", Body: "skill body"},
		},
	}}}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)

	// Skill written as a native file artifact.
	sk := findArtifact(arts, ".agents/skills/review-pr/SKILL.md")
	is.True(sk != nil)
	is.Equal(sk.Mode, target.ModeFile)
	is.True(strings.Contains(sk.Content, "name: review-pr"))
	is.True(strings.Contains(sk.Content, "description: review a PR"))
	is.True(strings.Contains(sk.Content, "skill body"))

	// Skills must NOT be inlined into .rules.
	blk := findArtifact(arts, ".rules")
	is.True(blk == nil || !strings.Contains(blk.Content, "## Skills"))
	is.True(blk == nil || !strings.Contains(blk.Content, "skill body"))
}

func TestGenerate_SkillOnlyProducesNoRulesArtifact(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Skills: []source.Primitive{
			{Kind: source.KindSkill, Name: "my-skill", Description: "does things", Body: "instructions"},
		},
	}}}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)

	// No .rules artifact when the only content is skills.
	is.True(findArtifact(arts, ".rules") == nil)
}

func TestGenerate_SkillFrontmatterFields(t *testing.T) {
	is := is.New(t)
	s := &source.Source{
		HatchVersion: "1.2.3",
		Scopes: []source.Scope{{
			Skills: []source.Primitive{
				{Kind: source.KindSkill, Name: "lint", Description: "run linters", Body: "lint instructions"},
			},
		}},
	}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)

	sk := findArtifact(arts, ".agents/skills/lint/SKILL.md")
	is.True(sk != nil)
	is.True(strings.Contains(sk.Content, "name: lint"))
	is.True(strings.Contains(sk.Content, "description: run linters"))
	is.True(strings.Contains(sk.Content, "hatch@1.2.3"))
}

func TestGenerate_SkillNestedScopeWritesToScopedPath(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: ""},
		{Path: "backend", Skills: []source.Primitive{
			{Kind: source.KindSkill, Name: "migrate", Description: "run migrations", Body: "migration steps"},
		}},
	}}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)

	sk := findArtifact(arts, "backend/.agents/skills/migrate/SKILL.md")
	is.True(sk != nil)
	is.True(strings.Contains(sk.Content, "name: migrate"))
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

func TestGenerate_SkillTargetFilteringExcludesUnselectedSkills(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Skills: []source.Primitive{
			{Kind: source.KindSkill, Name: "claude-only", Description: "claude skill", Body: "body", Targets: []string{"claude"}},
			{Kind: source.KindSkill, Name: "all-targets", Description: "universal skill", Body: "universal body"},
		},
	}}}
	arts, err := zed.New().Generate(s)
	is.NoErr(err)

	is.True(findArtifact(arts, ".agents/skills/all-targets/SKILL.md") != nil)
	is.True(findArtifact(arts, ".agents/skills/claude-only/SKILL.md") == nil)
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
