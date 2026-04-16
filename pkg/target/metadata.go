package target

import (
	"github.com/grafana/hatch/pkg/render"
	"gopkg.in/yaml.v3"
)

// MetadataField returns a frontmatter field containing hatch's
// generation metadata: which version of hatch produced this file
// (`generated: hatch@<version>`) and where the authoring source file
// lives (`source: .hatch/...`).
//
// The shape follows the agentskills.io `metadata` convention — a
// free-form string/string map nested under a top-level `metadata` key
// — so hatch's generated keys don't collide with spec-defined
// frontmatter fields like `license` or `compatibility`. Targets that
// aren't agentskills (commands, agents, Cursor .mdc, Copilot
// instructions) use the same shape for consistency.
//
// Either value is omitted if its input string is empty. If both are
// empty the returned Field has a nil Value so render.Frontmatter drops
// it entirely rather than writing an empty `metadata: {}` block.
func MetadataField(hatchVersion, sourcePath string) render.Field {
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	if hatchVersion != "" {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "generated"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "hatch@" + hatchVersion},
		)
	}
	if sourcePath != "" {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "source"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: sourcePath},
		)
	}
	if len(mapping.Content) == 0 {
		return render.Field{Key: "metadata", Value: nil}
	}
	return render.Field{Key: "metadata", Value: mapping}
}
