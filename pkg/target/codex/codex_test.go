package codex_test

import (
	"strings"
	"testing"

	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
	"github.com/grafana/hatch/pkg/target/codex"
	"github.com/matryer/is"
)

func TestGenerate_RulesAsBlockInAGENTS(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Rules: []source.Primitive{
			{Kind: source.KindRule, Name: "style", Body: "rule body"},
		},
	}}}
	arts, err := codex.New().Generate(s)
	is.NoErr(err)
	var blk *target.Artifact
	for i := range arts {
		if arts[i].Path == "AGENTS.md" {
			blk = &arts[i]
		}
	}
	is.True(blk != nil)
	is.Equal(blk.Mode, target.ModeBlock)
	is.True(strings.Contains(blk.Content, "rule body"))
}

func TestGenerate_SkillUsesAgentskillsPath(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Skills: []source.Primitive{{
			Kind: source.KindSkill, Name: "review-pr",
			Description: "d", Body: "b",
		}},
	}}}
	arts, err := codex.New().Generate(s)
	is.NoErr(err)
	found := false
	for _, a := range arts {
		if a.Path == ".agents/skills/review-pr/SKILL.md" {
			found = true
		}
	}
	is.True(found) // uses .agents/skills/ not .codex/skills/
}

func TestGenerate_SkillSKILLHasMetadata(t *testing.T) {
	is := is.New(t)
	s := &source.Source{
		HatchVersion: "v0.9.9-test",
		Scopes: []source.Scope{{
			Skills: []source.Primitive{{
				Kind: source.KindSkill, Name: "review-pr", Description: "d", Body: "b", SourcePath: "x",
			}},
		}},
	}
	arts, err := codex.New().Generate(s)
	is.NoErr(err)
	for _, a := range arts {
		if a.Path == ".agents/skills/review-pr/SKILL.md" {
			is.True(strings.Contains(a.Content, "metadata:"))
			is.True(strings.Contains(a.Content, "generated: hatch@v0.9.9-test"))
			is.True(strings.Contains(a.Content, "source: .hatch/_skills/review-pr/SKILL.md"))
			return
		}
	}
	t.Fatal("skill SKILL.md not found")
}

func TestGenerate_ScopedRulesGoToNestedAGENTS(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: ""},
		{
			Path: "backend",
			Rules: []source.Primitive{
				{Kind: source.KindRule, Name: "style", Body: "BACKEND CODEX RULE"},
			},
		},
	}}
	arts, err := codex.New().Generate(s)
	is.NoErr(err)
	var blk *target.Artifact
	for i := range arts {
		if arts[i].Path == "backend/AGENTS.md" {
			blk = &arts[i]
		}
	}
	is.True(blk != nil)
	is.Equal(blk.Mode, target.ModeBlock)
	is.True(strings.Contains(blk.Content, "BACKEND CODEX RULE"))
}

func TestGenerate_ScopedSkillUnderAgentsSkillsPath(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: ""},
		{
			Path: "backend",
			Skills: []source.Primitive{{
				Kind: source.KindSkill, Name: "review", Description: "d", Body: "b",
			}},
		},
	}}
	arts, err := codex.New().Generate(s)
	is.NoErr(err)
	found := false
	for _, a := range arts {
		if a.Path == "backend/.agents/skills/review/SKILL.md" {
			found = true
		}
	}
	is.True(found)
}

func TestGenerate_CommandsAndAgentsSkipped(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Commands: []source.Primitive{{Kind: source.KindCommand, Name: "c", Description: "d", Body: "b"}},
		Agents:   []source.Primitive{{Kind: source.KindAgent, Name: "a", Description: "d", Body: "b"}},
	}}}
	arts, err := codex.New().Generate(s)
	is.NoErr(err)
	for _, a := range arts {
		is.True(!strings.Contains(a.Path, "commands"))
		is.True(!strings.Contains(a.Path, "agents/"))
	}
}
