package opencode_test

import (
	"strings"
	"testing"

	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
	"github.com/grafana/hatch/pkg/target/opencode"
	"github.com/matryer/is"
	"gopkg.in/yaml.v3"
)

func ocMetaOverride(key, value string) map[string]*yaml.Node {
	return map[string]*yaml.Node{
		"opencode": {
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "metadata"},
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: key},
						{Kind: yaml.ScalarNode, Value: value},
					},
				},
			},
		},
	}
}

func TestGenerate_SkillMergesSourceMetadata(t *testing.T) {
	is := is.New(t)
	s := &source.Source{
		HatchVersion: "v0.9.9-test",
		Scopes: []source.Scope{{
			Skills: []source.Primitive{{
				Kind: source.KindSkill, Name: "review-pr", Description: "d", Body: "b",
				SourcePath: "x", Overrides: ocMetaOverride("author", "me"),
			}},
		}},
	}
	arts, err := opencode.New().Generate(s)
	is.NoErr(err)
	for _, a := range arts {
		if a.Path == ".opencode/skills/review-pr/SKILL.md" {
			is.Equal(strings.Count(a.Content, "metadata:"), 1)
			is.True(strings.Contains(a.Content, "author: me"))
			is.True(strings.Contains(a.Content, "generated: hatch@v0.9.9-test"))
			return
		}
	}
	t.Fatal("skill SKILL.md not found")
}

func TestGenerate_AllFourPrimitives(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Rules: []source.Primitive{
			{Kind: source.KindRule, Name: "style", Body: "rule body"},
		},
		Skills: []source.Primitive{
			{Kind: source.KindSkill, Name: "review-pr", Description: "d", Body: "b"},
		},
		Commands: []source.Primitive{
			{Kind: source.KindCommand, Name: "commit", Description: "d", Body: "b"},
		},
		Agents: []source.Primitive{
			{Kind: source.KindAgent, Name: "security", Description: "d", Body: "b"},
		},
	}}}
	arts, err := opencode.New().Generate(s)
	is.NoErr(err)

	paths := make(map[string]target.WriteMode)
	for _, a := range arts {
		paths[a.Path] = a.Mode
	}

	is.Equal(paths["AGENTS.md"], target.ModeBlock)
	is.Equal(paths[".opencode/skills/review-pr/SKILL.md"], target.ModeFile)
	is.Equal(paths[".opencode/commands/commit.md"], target.ModeFile)
	is.Equal(paths[".opencode/agents/security.md"], target.ModeFile)
}

func TestGenerate_NamespacedCommandFilenameFlattened(t *testing.T) {
	// OpenCode reads commands from .opencode/commands/<name>.md and has no
	// namespace convention. Source command "opsx/apply" flattens to
	// opsx-apply.md.
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Commands: []source.Primitive{{
			Kind: source.KindCommand, Name: "opsx/apply", Description: "d", Body: "b",
		}},
	}}}
	arts, err := opencode.New().Generate(s)
	is.NoErr(err)
	found := false
	for _, a := range arts {
		if a.Path == ".opencode/commands/opsx-apply.md" {
			found = true
		}
		if a.Path == ".opencode/commands/opsx/apply.md" {
			t.Fatalf("namespaced filename leaked through: %s", a.Path)
		}
	}
	is.True(found)
}

func TestGenerate_FrontmatterFilesHaveMetadata(t *testing.T) {
	is := is.New(t)
	s := &source.Source{
		HatchVersion: "v0.9.9-test",
		Scopes: []source.Scope{{
			Skills: []source.Primitive{{
				Kind: source.KindSkill, Name: "review-pr", Description: "d", Body: "b", SourcePath: "x",
			}},
			Commands: []source.Primitive{{
				Kind: source.KindCommand, Name: "commit", Description: "d", Body: "b", SourcePath: "x",
			}},
			Agents: []source.Primitive{{
				Kind: source.KindAgent, Name: "security", Description: "d", Body: "b", SourcePath: "x",
			}},
		}},
	}
	arts, err := opencode.New().Generate(s)
	is.NoErr(err)
	byPath := map[string]string{}
	for _, a := range arts {
		byPath[a.Path] = a.Content
	}
	skill := byPath[".opencode/skills/review-pr/SKILL.md"]
	is.True(strings.Contains(skill, "metadata:"))
	is.True(strings.Contains(skill, "generated: hatch@v0.9.9-test"))
	is.True(strings.Contains(skill, "source: .hatch/_skills/review-pr/SKILL.md"))
	cmd := byPath[".opencode/commands/commit.md"]
	is.True(strings.Contains(cmd, "source: .hatch/_commands/commit.md"))
	agent := byPath[".opencode/agents/security.md"]
	is.True(strings.Contains(agent, "source: .hatch/_agents/security.md"))
}

func TestGenerate_ScopedPrimitivesNested(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: ""},
		{
			Path: "backend",
			Rules: []source.Primitive{
				{Kind: source.KindRule, Name: "style", Body: "OC RULE"},
			},
			Skills: []source.Primitive{
				{Kind: source.KindSkill, Name: "review", Description: "d", Body: "b"},
			},
			Commands: []source.Primitive{
				{Kind: source.KindCommand, Name: "deploy", Description: "d", Body: "b"},
			},
			Agents: []source.Primitive{
				{Kind: source.KindAgent, Name: "guard", Description: "d", Body: "b"},
			},
		},
	}}
	arts, err := opencode.New().Generate(s)
	is.NoErr(err)
	byPath := map[string]target.WriteMode{}
	contentBy := map[string]string{}
	for _, a := range arts {
		byPath[a.Path] = a.Mode
		contentBy[a.Path] = a.Content
	}
	is.Equal(byPath["backend/AGENTS.md"], target.ModeBlock)
	is.True(strings.Contains(contentBy["backend/AGENTS.md"], "OC RULE"))
	is.Equal(byPath["backend/.opencode/skills/review/SKILL.md"], target.ModeFile)
	is.Equal(byPath["backend/.opencode/commands/deploy.md"], target.ModeFile)
	is.Equal(byPath["backend/.opencode/agents/guard.md"], target.ModeFile)
}
