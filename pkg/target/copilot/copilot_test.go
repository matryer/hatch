package copilot_test

import (
	"strings"
	"testing"

	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
	"github.com/grafana/hatch/pkg/target/copilot"
	"github.com/matryer/is"
)

func TestGenerate_UnscopedRulesBlockInCopilotInstructions(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Rules: []source.Primitive{
			{Kind: source.KindRule, Name: "style", Body: "unscoped body"},
		},
	}}}
	arts, err := copilot.New().Generate(s)
	is.NoErr(err)
	var blk *target.Artifact
	for i := range arts {
		if arts[i].Path == ".github/copilot-instructions.md" {
			blk = &arts[i]
		}
	}
	is.True(blk != nil)
	is.Equal(blk.Mode, target.ModeBlock)
	is.True(strings.Contains(blk.Content, "unscoped body"))
}

func TestGenerate_ScopedRuleBecomesInstructionsFile(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Rules: []source.Primitive{{
			Kind:    source.KindRule,
			Name:    "go-rules",
			ApplyTo: "**/*.go",
			Body:    "Go-only rule",
		}},
	}}}
	arts, err := copilot.New().Generate(s)
	is.NoErr(err)
	var inst *target.Artifact
	for i := range arts {
		if arts[i].Path == ".github/instructions/go-rules.instructions.md" {
			inst = &arts[i]
		}
	}
	is.True(inst != nil)
	is.Equal(inst.Mode, target.ModeFile)
	is.True(strings.Contains(inst.Content, "applyTo: '**/*.go'") ||
		strings.Contains(inst.Content, "applyTo: \"**/*.go\""))
	is.True(strings.Contains(inst.Content, "Go-only rule"))
}

func TestGenerate_SkillInlinedIntoCopilotInstructionsBlock(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Skills: []source.Primitive{{
			Kind:        source.KindSkill,
			Name:        "review-pr",
			Description: "review prs",
			Body:        "skill body",
		}},
	}}}
	arts, err := copilot.New().Generate(s)
	is.NoErr(err)
	var blk *target.Artifact
	for i := range arts {
		if arts[i].Path == ".github/copilot-instructions.md" {
			blk = &arts[i]
		}
	}
	is.True(blk != nil) // skill should be inlined as a block here
	is.True(strings.Contains(blk.Content, "review-pr"))
	is.True(strings.Contains(blk.Content, "skill body"))
}

func TestGenerate_CommandBecomesPromptFile(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Commands: []source.Primitive{{
			Kind:        source.KindCommand,
			Name:        "commit",
			Description: "commit",
			Body:        "body",
		}},
	}}}
	arts, err := copilot.New().Generate(s)
	is.NoErr(err)
	found := false
	for _, a := range arts {
		if a.Path == ".github/prompts/commit.prompt.md" {
			found = true
		}
	}
	is.True(found)
}

func TestGenerate_NamespacedCommandPromptFilenameFlattened(t *testing.T) {
	// Copilot has no namespace support on prompt files. A source command
	// named "opsx/apply" flattens to opsx-apply.prompt.md at the root.
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Commands: []source.Primitive{{
			Kind:        source.KindCommand,
			Name:        "opsx/apply",
			Description: "apply",
			Body:        "body",
		}},
	}}}
	arts, err := copilot.New().Generate(s)
	is.NoErr(err)
	found := false
	for _, a := range arts {
		if a.Path == ".github/prompts/opsx-apply.prompt.md" {
			found = true
		}
		if a.Path == ".github/prompts/opsx/apply.prompt.md" {
			t.Fatalf("namespaced filename leaked through: %s", a.Path)
		}
	}
	is.True(found)
}

func TestGenerate_FrontmatterFilesHaveMetadata(t *testing.T) {
	// Copilot's file-owned frontmatter outputs (scoped rules,
	// prompt-file commands, agent files) all carry metadata.generated
	// and metadata.source. The root-scope copilot-instructions block
	// is not a frontmatter file so it's excluded.
	is := is.New(t)
	s := &source.Source{
		HatchVersion: "v0.9.9-test",
		Scopes: []source.Scope{{
			Rules: []source.Primitive{{
				Kind: source.KindRule, Name: "go-rules", ApplyTo: "**/*.go", Body: "go body", SourcePath: "x",
			}},
			Commands: []source.Primitive{{
				Kind: source.KindCommand, Name: "commit", Description: "d", Body: "b", SourcePath: "x",
			}},
			Agents: []source.Primitive{{
				Kind: source.KindAgent, Name: "security", Description: "d", Body: "b", SourcePath: "x",
			}},
		}},
	}
	arts, err := copilot.New().Generate(s)
	is.NoErr(err)
	byPath := map[string]string{}
	for _, a := range arts {
		byPath[a.Path] = a.Content
	}
	rule := byPath[".github/instructions/go-rules.instructions.md"]
	is.True(strings.Contains(rule, "metadata:"))
	is.True(strings.Contains(rule, "generated: hatch@v0.9.9-test"))
	is.True(strings.Contains(rule, "source: .hatch/_rules/go-rules.md"))

	cmd := byPath[".github/prompts/commit.prompt.md"]
	is.True(strings.Contains(cmd, "source: .hatch/_commands/commit.md"))

	agent := byPath[".github/agents/security.agent.md"]
	is.True(strings.Contains(agent, "source: .hatch/_agents/security.md"))
}

func TestGenerate_ScopedRule_NoApplyTo_BecomesInstructionsFile(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: ""},
		{
			Path: "backend",
			Rules: []source.Primitive{{
				Kind: source.KindRule, Name: "style", Body: "Backend rule body.",
			}},
		},
	}}
	arts, err := copilot.New().Generate(s)
	is.NoErr(err)
	var inst *target.Artifact
	for i := range arts {
		if arts[i].Path == ".github/instructions/backend-style.instructions.md" {
			inst = &arts[i]
		}
	}
	is.True(inst != nil) // scoped rule emitted as a Copilot instructions file
	is.Equal(inst.Mode, target.ModeFile)
	is.True(strings.Contains(inst.Content, "applyTo:") &&
		strings.Contains(inst.Content, "backend/**"))
	is.True(strings.Contains(inst.Content, "Backend rule body."))
}

func TestGenerate_ScopedRule_ApplyToPrepended(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: ""},
		{
			Path: "backend",
			Rules: []source.Primitive{{
				Kind:    source.KindRule,
				Name:    "go-rules",
				ApplyTo: "*.go",
				Body:    "Go body.",
			}},
		},
	}}
	arts, err := copilot.New().Generate(s)
	is.NoErr(err)
	var inst *target.Artifact
	for i := range arts {
		if arts[i].Path == ".github/instructions/backend-go-rules.instructions.md" {
			inst = &arts[i]
		}
	}
	is.True(inst != nil)
	is.True(strings.Contains(inst.Content, "applyTo:") &&
		strings.Contains(inst.Content, "backend/*.go"))
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
		_, err := copilot.New().Generate(s)
		is.True(err != nil)
		is.True(strings.Contains(err.Error(), "bad"))
	}
}

func TestGenerate_ScopedSkill_InlinedAtRootWithScopedHeading(t *testing.T) {
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
	arts, err := copilot.New().Generate(s)
	is.NoErr(err)
	var blk *target.Artifact
	for i := range arts {
		if arts[i].Path == ".github/copilot-instructions.md" {
			blk = &arts[i]
		}
	}
	is.True(blk != nil)
	is.True(strings.Contains(blk.Content, "Skill: backend/review"))
	is.True(strings.Contains(blk.Content, "scoped skill body"))
}

func TestGenerate_ScopedCommand_PromptFilenameSlugged(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: ""},
		{
			Path: "services/api",
			Commands: []source.Primitive{{
				Kind: source.KindCommand, Name: "deploy", Description: "d", Body: "b",
			}},
		},
	}}
	arts, err := copilot.New().Generate(s)
	is.NoErr(err)
	found := false
	for _, a := range arts {
		if a.Path == ".github/prompts/services-api-deploy.prompt.md" {
			found = true
		}
	}
	is.True(found)
}

func TestGenerate_ScopedAgent_FilenameSlugged(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: ""},
		{
			Path: "backend",
			Agents: []source.Primitive{{
				Kind: source.KindAgent, Name: "guard", Description: "d", Body: "b",
			}},
		},
	}}
	arts, err := copilot.New().Generate(s)
	is.NoErr(err)
	found := false
	for _, a := range arts {
		if a.Path == ".github/agents/backend-guard.agent.md" {
			found = true
		}
	}
	is.True(found)
}

func TestGenerate_RootCopilotBlockUnchangedByScopedRules(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{
			Path: "",
			Rules: []source.Primitive{
				{Kind: source.KindRule, Name: "global", Body: "GLOBAL ROOT RULE"},
			},
		},
		{
			Path: "backend",
			Rules: []source.Primitive{
				{Kind: source.KindRule, Name: "scoped", Body: "BACKEND SCOPED RULE"},
			},
		},
	}}
	arts, err := copilot.New().Generate(s)
	is.NoErr(err)
	var blk *target.Artifact
	for i := range arts {
		if arts[i].Path == ".github/copilot-instructions.md" {
			blk = &arts[i]
		}
	}
	is.True(blk != nil)
	is.True(strings.Contains(blk.Content, "GLOBAL ROOT RULE"))
	is.True(!strings.Contains(blk.Content, "BACKEND SCOPED RULE"))
	// And there should be exactly one copilot-instructions.md across all scopes.
	count := 0
	for _, a := range arts {
		if a.Path == ".github/copilot-instructions.md" {
			count++
		}
	}
	is.Equal(count, 1)
}

func TestGenerate_AgentBecomesAgentFile(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Agents: []source.Primitive{{
			Kind:        source.KindAgent,
			Name:        "security",
			Description: "d",
			Body:        "body",
		}},
	}}}
	arts, err := copilot.New().Generate(s)
	is.NoErr(err)
	found := false
	for _, a := range arts {
		if a.Path == ".github/agents/security.agent.md" {
			found = true
		}
	}
	is.True(found)
}
