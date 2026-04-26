package target

import (
	"github.com/grafana/hatch/pkg/render"
	"gopkg.in/yaml.v3"
)

// MergeField folds a target-synthesized Field into fields, handling
// collisions with source-provided values sensibly:
//
//   - If the new Field's Value is nil, fields is returned unchanged
//     (lets the caller pass a no-op stamp without a guard).
//   - If fields has no Field with a matching Key, the new one is
//     appended verbatim.
//   - If the existing Field's Value is a YAML mapping and the new
//     Field's Value is also a YAML mapping, their contents are merged:
//     keys already present in the existing mapping are left alone
//     (source wins), and keys only in the new mapping are appended.
//   - If the existing Field's Value is anything other than a mapping,
//     the new Field is dropped. This preserves the source's intent
//     (even when malformed) and avoids emitting two top-level keys
//     with the same name — which would be invalid YAML.
//
// Intended use: after render.MergeOverride has folded a per-target
// passthrough block into fields, use MergeField to add hatch's own
// synthesized fields (e.g. MetadataField) without risking duplicate
// top-level keys when the passthrough already set the same key.
func MergeField(fields []render.Field, newField render.Field) []render.Field {
	if newField.Value == nil {
		return fields
	}
	for i, f := range fields {
		if f.Key != newField.Key {
			continue
		}
		existing, existingIsMap := f.Value.(*yaml.Node)
		incoming, incomingIsMap := newField.Value.(*yaml.Node)
		if !existingIsMap || existing == nil || existing.Kind != yaml.MappingNode {
			return fields
		}
		if !incomingIsMap || incoming == nil || incoming.Kind != yaml.MappingNode {
			return fields
		}
		fields[i].Value = mergeMappingPreferExisting(existing, incoming)
		return fields
	}
	return append(fields, newField)
}

// mergeMappingPreferExisting returns a mapping node whose keys are the
// union of dst and src. Where both sides have the same scalar key, dst
// wins — reflecting the "source overrides hatch defaults" precedence.
// Non-scalar keys in src are ignored (YAML mappings should key on
// scalars in practice).
func mergeMappingPreferExisting(dst, src *yaml.Node) *yaml.Node {
	have := map[string]bool{}
	for i := 0; i < len(dst.Content); i += 2 {
		k := dst.Content[i]
		if k.Kind == yaml.ScalarNode {
			have[k.Value] = true
		}
	}
	for i := 0; i < len(src.Content); i += 2 {
		k := src.Content[i]
		v := src.Content[i+1]
		if k.Kind != yaml.ScalarNode {
			continue
		}
		if have[k.Value] {
			continue
		}
		dst.Content = append(dst.Content, k, v)
		have[k.Value] = true
	}
	return dst
}
