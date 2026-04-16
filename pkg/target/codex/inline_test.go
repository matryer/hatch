package codex_test

import (
	"strings"
	"testing"

	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
	"github.com/grafana/hatch/pkg/target/codex"
	"github.com/matryer/is"
)

// Codex has no first-class slash commands or sub-agents. Rather than
// silently dropping those primitives, hatch inlines them into AGENTS.md
// as "here's how to respond if the user asks for this" sections so Codex
// still has access to the content.

func TestCodex_InlinesCommandsIntoAGENTSMd(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Commands: []source.Primitive{{
			Kind:        source.KindCommand,
			Name:        "commit",
			Description: "Commit current changes.",
			Body:        "Summarise the staged diff and create a commit.",
		}},
	}}}
	arts, err := codex.New().Generate(s)
	is.NoErr(err)

	var agents *target.Artifact
	for i := range arts {
		if arts[i].Path == "AGENTS.md" {
			agents = &arts[i]
		}
	}
	is.True(agents != nil) // AGENTS.md must be generated
	is.Equal(agents.Mode, target.ModeBlock)
	is.True(strings.Contains(agents.Content, "commit"))
	is.True(strings.Contains(agents.Content, "Commit current changes."))
	is.True(strings.Contains(agents.Content, "Summarise the staged diff"))
}

func TestCodex_InlinesAgentsIntoAGENTSMd(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Agents: []source.Primitive{{
			Kind:        source.KindAgent,
			Name:        "security-auditor",
			Description: "Review code for security issues.",
			Body:        "Focus on OWASP Top 10.",
		}},
	}}}
	arts, err := codex.New().Generate(s)
	is.NoErr(err)

	var agents *target.Artifact
	for i := range arts {
		if arts[i].Path == "AGENTS.md" {
			agents = &arts[i]
		}
	}
	is.True(agents != nil)
	is.True(strings.Contains(agents.Content, "security-auditor"))
	is.True(strings.Contains(agents.Content, "Review code for security issues."))
	is.True(strings.Contains(agents.Content, "Focus on OWASP Top 10."))
}

func TestCodex_RulesCommandsAgentsCombinedInOneBlock(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Rules: []source.Primitive{{
			Kind: source.KindRule, Name: "style", Body: "STYLE RULE",
		}},
		Commands: []source.Primitive{{
			Kind: source.KindCommand, Name: "commit", Description: "d", Body: "COMMAND BODY",
		}},
		Agents: []source.Primitive{{
			Kind: source.KindAgent, Name: "sec", Description: "d", Body: "AGENT BODY",
		}},
	}}}
	arts, err := codex.New().Generate(s)
	is.NoErr(err)

	var agents *target.Artifact
	for i := range arts {
		if arts[i].Path == "AGENTS.md" {
			agents = &arts[i]
		}
	}
	is.True(agents != nil)
	is.True(strings.Contains(agents.Content, "STYLE RULE"))
	is.True(strings.Contains(agents.Content, "COMMAND BODY"))
	is.True(strings.Contains(agents.Content, "AGENT BODY"))
}

func TestCodex_CommandsRespectTargetsFilter(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Commands: []source.Primitive{{
			Kind:        source.KindCommand,
			Name:        "opencode-only",
			Description: "d",
			Body:        "HIDDEN FROM CODEX",
			Targets:     []string{"opencode"},
		}},
	}}}
	arts, err := codex.New().Generate(s)
	is.NoErr(err)
	for _, a := range arts {
		is.True(!strings.Contains(a.Content, "HIDDEN FROM CODEX"))
	}
}
