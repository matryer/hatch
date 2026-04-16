package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/grafana/hatch/pkg/source"
)

// maxSlugLength bounds the filesystem name derived from a title so the
// resulting path stays well under any OS limit and remains legible.
const maxSlugLength = 60

// cmdNew scaffolds a single hatch source file for a given kind.
//
// Usage:
//
//	hatch new <kind> [title]
//
// kind is one of rule, skill, command, agent. If title is omitted, the
// user is prompted on stdin. The title is slugged into a filesystem-safe
// name and the appropriate template is written under .hatch/. After
// writing, the user is reminded to run `hatch gen`.
func cmdNew(_ context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Consume the kind first so the remaining args can carry the -path
	// flag in any order — Go's flag.Parse stops at the first non-flag
	// argument, so we can't put a positional `kind` before flags otherwise.
	if len(args) == 0 {
		return errors.New("hatch new: missing kind (one of rule, skill, command, agent)")
	}
	kind := args[0]
	tmpl, ok := templates[kind]
	if !ok {
		return fmt.Errorf("hatch new: unknown kind %q (want rule, skill, command, or agent)", kind)
	}

	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(stderr)
	pathFlag := fs.String("path", "", "create the new primitive under a nested scope path (e.g. backend or services/api)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	scopePath, err := validatePathFlag(*pathFlag)
	if err != nil {
		return err
	}
	positional := fs.Args()

	var title string
	if len(positional) > 0 {
		title = strings.Join(positional, " ")
	} else {
		prompted, err := promptTitle(kind, stdin, stdout)
		if err != nil {
			return err
		}
		title = prompted
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("hatch new: title is required")
	}

	slug := slugify(title)
	if slug == "" {
		return fmt.Errorf("hatch new: %q does not contain any slug characters", title)
	}

	srcRoot := ".hatch"
	if scopePath != "" {
		srcRoot = filepath.Join(srcRoot, filepath.FromSlash(scopePath))
	}
	path := filepath.Join(srcRoot, tmpl.pathOf(slug))
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("hatch new: %s already exists", path)
	}

	body := tmpl.render(title)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "created %s\n", path)
	fmt.Fprintln(stdout, "edit the file, then run `hatch gen` to write the output files.")
	return nil
}

// newTemplate knows how to place and render a source file for one
// primitive kind.
type newTemplate struct {
	pathOf func(slug string) string
	render func(title string) string
}

var templates = map[string]newTemplate{
	"rule": {
		pathOf: func(slug string) string { return filepath.Join(source.RulesDir, slug+".md") },
		render: func(title string) string {
			return fmt.Sprintf("# %s\n\nDescribe the rule here.\n", title)
		},
	},
	"skill": {
		pathOf: func(slug string) string { return filepath.Join(source.SkillsDir, slug, "SKILL.md") },
		render: func(title string) string {
			return fmt.Sprintf(`---
description: TODO — one-sentence description of what this skill does.
---

# %s

Describe the skill body here.
`, title)
		},
	},
	"command": {
		pathOf: func(slug string) string { return filepath.Join(source.CommandsDir, slug+".md") },
		render: func(title string) string {
			return fmt.Sprintf(`---
description: TODO — one-sentence description of what this command does.
---

# %s

Describe the command body here.
`, title)
		},
	},
	"agent": {
		pathOf: func(slug string) string { return filepath.Join(source.AgentsDir, slug+".md") },
		render: func(title string) string {
			return fmt.Sprintf(`---
description: TODO — one-sentence description of what this agent does.
---

# %s

Describe the agent's role and instructions here.
`, title)
		},
	},
}

// promptTitle reads one line from stdin after printing a "<kind> name: "
// prompt. Returns the trimmed input.
func promptTitle(kind string, stdin io.Reader, stdout io.Writer) (string, error) {
	if stdin == nil {
		return "", errors.New("hatch new: no title argument and no input available")
	}
	fmt.Fprintf(stdout, "%s name: ", kind)
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// slugify converts a free-form title into a filesystem-safe slug:
// lowercase ASCII letters/digits separated by single hyphens, trimmed of
// leading/trailing hyphens, and truncated to maxSlugLength characters.
// Non-ASCII characters are dropped.
func slugify(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.TrimSpace(s) {
		lower := unicode.ToLower(r)
		switch {
		case (lower >= 'a' && lower <= 'z') || (lower >= '0' && lower <= '9'):
			b.WriteRune(lower)
			lastDash = false
		default:
			if b.Len() > 0 && !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	out := strings.TrimSuffix(b.String(), "-")
	if len(out) > maxSlugLength {
		out = out[:maxSlugLength]
		out = strings.TrimSuffix(out, "-")
	}
	return out
}
