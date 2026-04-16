package target_test

import (
	"testing"

	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
	"github.com/matryer/is"
)

func TestSourceFilePath(t *testing.T) {
	cases := []struct {
		name  string
		scope string
		kind  source.Kind
		pName string
		want  string
	}{
		{"rule root", "", source.KindRule, "style", ".hatch/_rules/style.md"},
		{"rule scoped", "backend", source.KindRule, "style", ".hatch/backend/_rules/style.md"},
		{"rule deep scope", "services/api", source.KindRule, "r", ".hatch/services/api/_rules/r.md"},
		{"skill root", "", source.KindSkill, "review-pr", ".hatch/_skills/review-pr/SKILL.md"},
		{"skill scoped", "backend", source.KindSkill, "review", ".hatch/backend/_skills/review/SKILL.md"},
		{"command root flat", "", source.KindCommand, "commit", ".hatch/_commands/commit.md"},
		{"command root namespaced", "", source.KindCommand, "opsx/apply", ".hatch/_commands/opsx/apply.md"},
		{"command scoped namespaced", "backend", source.KindCommand, "opsx/apply", ".hatch/backend/_commands/opsx/apply.md"},
		{"agent root", "", source.KindAgent, "security", ".hatch/_agents/security.md"},
		{"agent scoped", "backend", source.KindAgent, "guard", ".hatch/backend/_agents/guard.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			got := target.SourceFilePath(tc.scope, tc.kind, tc.pName)
			is.Equal(got, tc.want)
		})
	}
}
