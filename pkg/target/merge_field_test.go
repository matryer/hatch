package target_test

import (
	"strings"
	"testing"

	"github.com/grafana/hatch/pkg/render"
	"github.com/grafana/hatch/pkg/target"
	"github.com/matryer/is"
	"gopkg.in/yaml.v3"
)

// mappingNode builds a YAML mapping from flat "k,v,k,v" pairs for
// terser test setup than constructing yaml.Node literals.
func mappingNode(pairs ...string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.MappingNode}
	for i := 0; i < len(pairs); i += 2 {
		n.Content = append(n.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: pairs[i]},
			&yaml.Node{Kind: yaml.ScalarNode, Value: pairs[i+1]},
		)
	}
	return n
}

func TestMergeField_NilValueIsNoOp(t *testing.T) {
	is := is.New(t)
	in := []render.Field{{Key: "name", Value: "x"}}
	out := target.MergeField(in, render.Field{Key: "metadata", Value: nil})
	is.Equal(len(out), 1)
	is.Equal(out[0].Key, "name")
}

func TestMergeField_AppendsWhenKeyAbsent(t *testing.T) {
	is := is.New(t)
	in := []render.Field{{Key: "name", Value: "x"}}
	stamp := render.Field{Key: "metadata", Value: mappingNode("generated", "hatch@v1")}
	out := target.MergeField(in, stamp)
	is.Equal(len(out), 2)
	is.Equal(out[1].Key, "metadata")
}

func TestMergeField_MergesMappingsPreservingExisting(t *testing.T) {
	// Existing metadata carries `author: me` (from a source `claude:`
	// passthrough). MergeField must nest hatch's keys under the same
	// metadata, not append a duplicate.
	is := is.New(t)
	existing := mappingNode("author", "me")
	in := []render.Field{
		{Key: "name", Value: "x"},
		{Key: "metadata", Value: existing},
	}
	stamp := render.Field{Key: "metadata", Value: mappingNode("generated", "hatch@v1", "source", ".hatch/x.md")}
	out := target.MergeField(in, stamp)
	is.Equal(len(out), 2) // still 2: no duplicate metadata field

	rendered, err := render.Frontmatter(out)
	is.NoErr(err)
	is.Equal(strings.Count(rendered, "metadata:"), 1)
	is.True(strings.Contains(rendered, "author: me"))
	is.True(strings.Contains(rendered, "generated: hatch@v1"))
	is.True(strings.Contains(rendered, "source: .hatch/x.md"))
}

func TestMergeField_SourceKeyWinsOnCollision(t *testing.T) {
	// If the source's metadata already has `source: override.md`,
	// hatch's attempt to set `source: .hatch/real.md` must not replace
	// the user's value.
	is := is.New(t)
	existing := mappingNode("source", "override.md")
	in := []render.Field{{Key: "metadata", Value: existing}}
	stamp := render.Field{Key: "metadata", Value: mappingNode("generated", "hatch@v1", "source", ".hatch/real.md")}
	out := target.MergeField(in, stamp)

	rendered, err := render.Frontmatter(out)
	is.NoErr(err)
	is.True(strings.Contains(rendered, "source: override.md"))
	is.True(!strings.Contains(rendered, "source: .hatch/real.md"))
	is.True(strings.Contains(rendered, "generated: hatch@v1")) // non-colliding key still added
}

func TestMergeField_NonMappingExistingKeptNoDuplicate(t *testing.T) {
	// A source that puts `metadata: "a-string"` in frontmatter is
	// malformed per agentskills, but hatch must not emit duplicate
	// top-level keys. Keep the source's weird value and skip hatch's
	// stamp; a future lint pass can surface the authoring mistake.
	is := is.New(t)
	scalar := &yaml.Node{Kind: yaml.ScalarNode, Value: "a-string"}
	in := []render.Field{{Key: "metadata", Value: scalar}}
	stamp := render.Field{Key: "metadata", Value: mappingNode("generated", "hatch@v1")}
	out := target.MergeField(in, stamp)
	is.Equal(len(out), 1)
	is.Equal(out[0].Value, scalar)

	rendered, err := render.Frontmatter(out)
	is.NoErr(err)
	is.Equal(strings.Count(rendered, "metadata:"), 1)
}
