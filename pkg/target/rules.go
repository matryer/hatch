package target

import (
	"strings"

	"github.com/grafana/hatch/pkg/render"
	"github.com/grafana/hatch/pkg/source"
)

// RulesBlock returns the concatenated markdown body for every rule that
// applies to `targetName`, ready to go inside a hatch block. Rules with an
// `applyTo` glob get a heading that includes the glob so the scope is visible
// in the generated file; otherwise the rule body is used verbatim (with any
// leading H1 already in the body preserved as-is).
func RulesBlock(sc *source.Scope, targetName, displayName string) string {
	var buf strings.Builder
	first := true
	for _, r := range sc.Rules {
		if !r.HasTarget(targetName) {
			continue
		}
		body := strings.TrimSpace(render.Body(r.Body, displayName, targetName))
		if body == "" {
			continue
		}
		if !first {
			buf.WriteString("\n\n")
		}
		first = false
		if r.ApplyTo != "" {
			buf.WriteString("## ")
			buf.WriteString(titleFromName(r.Name))
			buf.WriteString(" (applies to `")
			buf.WriteString(r.ApplyTo)
			buf.WriteString("`)\n\n")
		}
		buf.WriteString(body)
	}
	return buf.String()
}

// SkillsBlock returns skill bodies as sub-sections for use inside a hatch
// block (currently only used by the Copilot target, which has no native
// skill primitive).
func SkillsBlock(sc *source.Scope, targetName, displayName string) string {
	return inlinePrimitives(sc.Skills, "Skills", "Skill", targetName, displayName,
		"The following capabilities describe actions the user may ask the model to perform.")
}

// CommandsBlock returns command bodies as an inlined section for targets
// that have no native slash-command primitive (Codex). The section header
// explains how the agent should interpret a user's request for a command.
func CommandsBlock(sc *source.Scope, targetName, displayName string) string {
	return inlinePrimitives(sc.Commands, "Commands", "Command", targetName, displayName,
		"If the user asks to run one of these commands, follow the matching instructions below.")
}

// AgentsBlock returns agent bodies as an inlined section for targets that
// have no native sub-agent primitive (Codex). The section header explains
// how the model should behave when asked to delegate to one.
func AgentsBlock(sc *source.Scope, targetName, displayName string) string {
	return inlinePrimitives(sc.Agents, "Sub-agents", "Sub-agent", targetName, displayName,
		"If the user asks to delegate to one of these sub-agents, take on that role and follow the matching instructions.")
}

// inlinePrimitives renders a list of primitives as a markdown section with
// a top-level heading, a short intro sentence, and one sub-section per
// primitive. Returns an empty string if no primitives apply to targetName.
func inlinePrimitives(items []source.Primitive, sectionTitle, itemLabel, targetName, displayName, intro string) string {
	var filtered []source.Primitive
	for _, p := range items {
		if p.HasTarget(targetName) {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("## ")
	buf.WriteString(sectionTitle)
	buf.WriteString("\n\n")
	buf.WriteString(intro)
	buf.WriteString("\n")
	for _, p := range filtered {
		body := strings.TrimSpace(render.Body(p.Body, displayName, targetName))
		buf.WriteString("\n### ")
		buf.WriteString(itemLabel)
		buf.WriteString(": ")
		buf.WriteString(FlatName(p.Name))
		buf.WriteString("\n\n")
		if p.Description != "" {
			buf.WriteString("_")
			buf.WriteString(p.Description)
			buf.WriteString("_\n\n")
		}
		buf.WriteString(body)
		buf.WriteString("\n")
	}
	return strings.TrimRight(buf.String(), "\n")
}

// FlatName collapses a namespaced name like "opsx/apply" into a flat
// slug-style identifier "opsx-apply". Hatch's source model allows
// namespaced commands because Claude Code renders a subdirectory under
// .claude/commands/ as a colon-separated slash command (/opsx:apply).
// No other supported target speaks namespaces, so they flatten the
// slash to a dash both in filenames and in any textual identifier the
// agent might match on.
func FlatName(name string) string {
	return strings.ReplaceAll(name, "/", "-")
}

// titleFromName turns "coding-style" into "Coding style".
func titleFromName(name string) string {
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
