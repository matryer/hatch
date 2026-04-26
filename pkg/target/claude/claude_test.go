package claude_test

import (
	"strings"
	"testing"

	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
	"github.com/grafana/hatch/pkg/target/claude"
	"github.com/matryer/is"
	"gopkg.in/yaml.v3"
)

// mapNode builds a single-level yaml mapping for test fixtures.
func mapNode(pairs ...string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.MappingNode}
	for i := 0; i < len(pairs); i += 2 {
		n.Content = append(n.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: pairs[i]},
			&yaml.Node{Kind: yaml.ScalarNode, Value: pairs[i+1]},
		)
	}
	return n
}

// nestedMap wraps `pairs` in a mapping under the outer key. Used to
// simulate a source primitive's per-target passthrough:
// `claude: { metadata: { author: me } }` stores a mapping of
// {metadata: {author: me}} in Overrides["claude"].
func nestedMap(outer string, inner *yaml.Node) *yaml.Node {
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: outer},
			inner,
		},
	}
}

func TestGenerate_RulesAsBlockInCLAUDE(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Rules: []source.Primitive{
			{Kind: source.KindRule, Name: "coding-style", Body: "Write clean code."},
		},
	}}}
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	var blk *target.Artifact
	for i := range arts {
		if arts[i].Path == "CLAUDE.md" {
			blk = &arts[i]
			break
		}
	}
	is.True(blk != nil)                  // CLAUDE.md artifact produced
	is.Equal(blk.Mode, target.ModeBlock) // block-injected
	is.True(strings.Contains(blk.Content, "Write clean code."))
}

func TestGenerate_ApplyToRuleGetsHeading(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Rules: []source.Primitive{
			{Kind: source.KindRule, Name: "go-rules", ApplyTo: "**/*.go", Body: "Go body."},
		},
	}}}
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	var blk *target.Artifact
	for i := range arts {
		if arts[i].Path == "CLAUDE.md" {
			blk = &arts[i]
		}
	}
	is.True(blk != nil)
	is.True(strings.Contains(blk.Content, "## Go rules (applies to `**/*.go`)"))
	is.True(strings.Contains(blk.Content, "Go body."))
}

func TestGenerate_AddsMetadataBlock(t *testing.T) {
	// Every file-owned frontmatter output carries a metadata block with
	// `generated: hatch@<version>` and `source: .hatch/...`. This test
	// covers a skill; a command (which reuses renderSkill) is covered
	// implicitly, but we also assert an agent and a scoped command to
	// pin the source path shape for each kind and scope.
	is := is.New(t)
	s := &source.Source{
		HatchVersion: "v0.9.9-test",
		Scopes: []source.Scope{
			{
				Path: "",
				Skills: []source.Primitive{{
					Kind: source.KindSkill, Name: "review-pr", Description: "d", Body: "b", SourcePath: "x",
				}},
				Commands: []source.Primitive{{
					Kind: source.KindCommand, Name: "opsx/apply", Description: "d", Body: "b", SourcePath: "x",
				}},
				Agents: []source.Primitive{{
					Kind: source.KindAgent, Name: "guard", Description: "d", Body: "b", SourcePath: "x",
				}},
			},
			{
				Path: "backend",
				Commands: []source.Primitive{{
					Kind: source.KindCommand, Name: "deploy", Description: "d", Body: "b", SourcePath: "x",
				}},
			},
		},
	}
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	byPath := map[string]string{}
	for _, a := range arts {
		byPath[a.Path] = a.Content
	}
	skill := byPath[".claude/skills/review-pr/SKILL.md"]
	is.True(strings.Contains(skill, "metadata:"))
	is.True(strings.Contains(skill, "generated: hatch@v0.9.9-test"))
	is.True(strings.Contains(skill, "source: .hatch/_skills/review-pr/SKILL.md"))

	cmd := byPath[".claude/commands/opsx/apply.md"]
	is.True(strings.Contains(cmd, "source: .hatch/_commands/opsx/apply.md"))

	agent := byPath[".claude/agents/guard.md"]
	is.True(strings.Contains(agent, "source: .hatch/_agents/guard.md"))

	scoped := byPath["backend/.claude/commands/deploy.md"]
	is.True(strings.Contains(scoped, "source: .hatch/backend/_commands/deploy.md"))
}

func TestGenerate_MergesHatchMetadataIntoSourceMetadata(t *testing.T) {
	// Regression for the v0.4.0 bug: when a source skill carries a
	// per-target `claude: { metadata: { author: me } }` passthrough,
	// hatch's MetadataField must merge INTO that existing metadata
	// mapping, not emit a second top-level `metadata:` key (invalid
	// YAML).
	is := is.New(t)
	claudeOverride := nestedMap("metadata", mapNode("author", "me", "version", "1.0"))
	s := &source.Source{
		HatchVersion: "v0.9.9-test",
		Scopes: []source.Scope{{
			Skills: []source.Primitive{{
				Kind:        source.KindSkill,
				Name:        "review-pr",
				Description: "d",
				Body:        "b",
				SourcePath:  "x",
				Overrides:   map[string]*yaml.Node{"claude": claudeOverride},
			}},
		}},
	}
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	for _, a := range arts {
		if a.Path != ".claude/skills/review-pr/SKILL.md" {
			continue
		}
		is.Equal(strings.Count(a.Content, "metadata:"), 1) // no duplicate key
		is.True(strings.Contains(a.Content, "author: me"))
		is.True(strings.Contains(a.Content, "version:"))
		is.True(strings.Contains(a.Content, "generated: hatch@v0.9.9-test"))
		is.True(strings.Contains(a.Content, "source: .hatch/_skills/review-pr/SKILL.md"))
		return
	}
	t.Fatal("SKILL.md artifact not found")
}

func TestGenerate_SourceMetadataKeyWinsOnCollision(t *testing.T) {
	// If the source's claude-metadata sets `source: overridden.md`,
	// hatch must not replace it with the real .hatch/ path.
	is := is.New(t)
	claudeOverride := nestedMap("metadata", mapNode("source", "overridden.md"))
	s := &source.Source{
		HatchVersion: "v0.9.9-test",
		Scopes: []source.Scope{{
			Skills: []source.Primitive{{
				Kind: source.KindSkill, Name: "review-pr", Description: "d", Body: "b", SourcePath: "x",
				Overrides: map[string]*yaml.Node{"claude": claudeOverride},
			}},
		}},
	}
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	for _, a := range arts {
		if a.Path != ".claude/skills/review-pr/SKILL.md" {
			continue
		}
		is.True(strings.Contains(a.Content, "source: overridden.md"))
		is.True(!strings.Contains(a.Content, "source: .hatch/_skills/review-pr/SKILL.md"))
		is.True(strings.Contains(a.Content, "generated: hatch@v0.9.9-test")) // non-colliding still added
		return
	}
	t.Fatal("SKILL.md artifact not found")
}

func TestGenerate_SkillBecomesFileUnderDotClaude(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Skills: []source.Primitive{{
			Kind:        source.KindSkill,
			Name:        "review-pr",
			Description: "Review a PR.",
			Body:        "do stuff\n",
		}},
	}}}
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	var skill *target.Artifact
	for i := range arts {
		if strings.HasSuffix(arts[i].Path, "review-pr/SKILL.md") {
			skill = &arts[i]
			break
		}
	}
	is.True(skill != nil)
	is.Equal(skill.Mode, target.ModeFile)
	is.True(strings.HasPrefix(skill.Content, "---\n"))
	is.True(strings.Contains(skill.Content, "name: review-pr"))
	is.True(strings.Contains(skill.Content, "description: Review a PR."))
	is.True(strings.Contains(skill.Content, "do stuff"))
}

func TestGenerate_CommandBecomesDotClaudeCommand(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Commands: []source.Primitive{{
			Kind:        source.KindCommand,
			Name:        "commit",
			Description: "Commit with a generated message.",
			Body:        "body\n",
		}},
	}}}
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	found := false
	for _, a := range arts {
		if a.Path == ".claude/commands/commit.md" {
			found = true
			is.Equal(a.Mode, target.ModeFile)
		}
	}
	is.True(found)
}

func TestGenerate_NamespacedCommandPreservesSubdirectory(t *testing.T) {
	// Claude Code reads .claude/commands/<ns>/<name>.md and presents it as
	// the slash command /<ns>:<name>. The hatch source name carries the
	// forward slash ("opsx/apply") and Claude is the one target that
	// preserves that subdirectory verbatim; other targets flatten it.
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Commands: []source.Primitive{{
			Kind:        source.KindCommand,
			Name:        "opsx/apply",
			Description: "Apply an opsx change.",
			Body:        "body\n",
		}},
	}}}
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	found := false
	for _, a := range arts {
		if a.Path == ".claude/commands/opsx/apply.md" {
			found = true
			is.Equal(a.Mode, target.ModeFile)
			is.True(strings.Contains(a.Content, "name: opsx/apply"))
		}
	}
	is.True(found)
}

func TestGenerate_AgentBecomesDotClaudeAgent(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Agents: []source.Primitive{{
			Kind:        source.KindAgent,
			Name:        "security-auditor",
			Description: "Security review.",
			Body:        "body\n",
		}},
	}}}
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	found := false
	for _, a := range arts {
		if a.Path == ".claude/agents/security-auditor.md" {
			found = true
		}
	}
	is.True(found)
}

func TestGenerate_ScopedRulesGoToNestedCLAUDE(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: ""},
		{
			Path: "backend",
			Rules: []source.Primitive{
				{Kind: source.KindRule, Name: "style", Body: "Backend style."},
			},
		},
	}}
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	var blk *target.Artifact
	for i := range arts {
		if arts[i].Path == "backend/CLAUDE.md" {
			blk = &arts[i]
		}
	}
	is.True(blk != nil) // backend/CLAUDE.md artifact produced
	is.Equal(blk.Mode, target.ModeBlock)
	is.True(strings.Contains(blk.Content, "Backend style."))
}

func TestGenerate_ScopedSkillToNestedClaudeDir(t *testing.T) {
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
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	found := false
	for _, a := range arts {
		if a.Path == "backend/.claude/skills/review/SKILL.md" {
			found = true
			is.Equal(a.Mode, target.ModeFile)
		}
	}
	is.True(found)
}

func TestGenerate_ScopedCommandAndAgentNested(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{Path: ""},
		{
			Path: "backend",
			Commands: []source.Primitive{{
				Kind: source.KindCommand, Name: "deploy", Description: "d", Body: "b",
			}},
			Agents: []source.Primitive{{
				Kind: source.KindAgent, Name: "guard", Description: "d", Body: "b",
			}},
		},
	}}
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	paths := map[string]bool{}
	for _, a := range arts {
		paths[a.Path] = true
	}
	is.True(paths["backend/.claude/commands/deploy.md"])
	is.True(paths["backend/.claude/agents/guard.md"])
}

func TestGenerate_RootAndNestedCLAUDECoexist(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{
		{
			Path: "",
			Rules: []source.Primitive{
				{Kind: source.KindRule, Name: "global", Body: "GLOBAL RULE"},
			},
		},
		{
			Path: "backend",
			Rules: []source.Primitive{
				{Kind: source.KindRule, Name: "api", Body: "BACKEND RULE"},
			},
		},
	}}
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	byPath := map[string]string{}
	for _, a := range arts {
		byPath[a.Path] = a.Content
	}
	is.True(strings.Contains(byPath["CLAUDE.md"], "GLOBAL RULE"))
	is.True(!strings.Contains(byPath["CLAUDE.md"], "BACKEND RULE"))
	is.True(strings.Contains(byPath["backend/CLAUDE.md"], "BACKEND RULE"))
	is.True(!strings.Contains(byPath["backend/CLAUDE.md"], "GLOBAL RULE"))
}

func TestGenerate_RespectsTargetsFilter(t *testing.T) {
	is := is.New(t)
	s := &source.Source{Scopes: []source.Scope{{
		Skills: []source.Primitive{{
			Kind:        source.KindSkill,
			Name:        "only-opencode",
			Description: "d",
			Body:        "b",
			Targets:     []string{"opencode"},
		}},
	}}}
	arts, err := claude.New().Generate(s)
	is.NoErr(err)
	// The only skill opts out of claude — no artifacts expected.
	is.Equal(len(arts), 0)
}
