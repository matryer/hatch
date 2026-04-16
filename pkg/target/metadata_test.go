package target_test

import (
	"strings"
	"testing"

	"github.com/grafana/hatch/pkg/render"
	"github.com/grafana/hatch/pkg/target"
	"github.com/matryer/is"
)

func TestMetadataField_BothKeys(t *testing.T) {
	is := is.New(t)
	f := target.MetadataField("v0.3.0", ".hatch/_skills/review/SKILL.md")
	is.Equal(f.Key, "metadata")
	out, err := render.Frontmatter([]render.Field{f})
	is.NoErr(err)
	is.True(strings.Contains(out, "metadata:"))
	is.True(strings.Contains(out, "generated: hatch@v0.3.0"))
	is.True(strings.Contains(out, "source: .hatch/_skills/review/SKILL.md"))
}

func TestMetadataField_OmitsGeneratedWhenVersionEmpty(t *testing.T) {
	is := is.New(t)
	f := target.MetadataField("", ".hatch/_rules/style.md")
	out, err := render.Frontmatter([]render.Field{f})
	is.NoErr(err)
	is.True(strings.Contains(out, "source: .hatch/_rules/style.md"))
	is.True(!strings.Contains(out, "generated:"))
}

func TestMetadataField_EmptyInputProducesNoField(t *testing.T) {
	// When both inputs are empty, the field should drop out entirely so
	// we don't render an empty `metadata: {}` block.
	is := is.New(t)
	f := target.MetadataField("", "")
	out, err := render.Frontmatter([]render.Field{f})
	is.NoErr(err)
	is.Equal(out, "")
}
