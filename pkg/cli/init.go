package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/grafana/hatch/pkg/source"
)

// cmdInit scaffolds `.hatch/` with the four primitive container
// subdirectories. By default the directories are empty; `-examples`
// additionally writes one example primitive of each kind so a new user
// can `hatch gen` immediately and see output. `-path <relative-path>`
// scaffolds the dirs under a nested scope (e.g. .hatch/backend/_rules/).
func cmdInit(_ context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	examples := fs.Bool("examples", false, "also write one example rule, skill, command, and agent")
	pathFlag := fs.String("path", "", "scaffold under a nested scope path (e.g. backend or services/api)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	scopePath, err := validatePathFlag(*pathFlag)
	if err != nil {
		return err
	}
	srcRoot := ".hatch"
	if scopePath != "" {
		srcRoot = filepath.Join(srcRoot, filepath.FromSlash(scopePath))
	}
	for _, sub := range []string{source.RulesDir, source.SkillsDir, source.CommandsDir, source.AgentsDir} {
		if err := os.MkdirAll(filepath.Join(srcRoot, sub), 0o755); err != nil {
			return err
		}
	}
	if !*examples {
		fmt.Fprintf(stdout, "created %s/ (run `hatch new <kind> <title>` to add source files, or `hatch init -examples` for a starter set)\n", srcRoot)
		return nil
	}

	// Files in a slice so iteration order (and therefore stdout output) is
	// deterministic across runs. Map iteration in Go is randomized.
	type seed struct {
		path string
		body string
	}
	seeds := []seed{
		{
			path: filepath.Join(srcRoot, source.RulesDir, "coding-style.md"),
			body: "Write clean, well-tested Go. Prefer short functions. Use table-driven tests.\n",
		},
		{
			path: filepath.Join(srcRoot, source.SkillsDir, "review-pr", "SKILL.md"),
			body: `---
description: Review an open pull request for correctness, style, and tests.
---

# review-pr

When asked to review a PR, check out the branch, read the diff end-to-end,
and report findings as: (1) bugs, (2) style nits, (3) missing tests.
`,
		},
		{
			path: filepath.Join(srcRoot, source.CommandsDir, "commit.md"),
			body: `---
description: Stage and commit current changes with a generated message.
---

# commit

Summarise the staged diff in one sentence and create a commit with that
message.
`,
		},
		{
			path: filepath.Join(srcRoot, source.AgentsDir, "security-auditor.md"),
			body: `---
description: Review code for common security pitfalls (injection, XSS, auth).
---

# security-auditor

Focus on OWASP Top 10 categories. Report findings as file:line references.
`,
		},
	}
	sort.Slice(seeds, func(i, j int) bool { return seeds[i].path < seeds[j].path })

	for _, s := range seeds {
		if _, err := os.Stat(s.path); err == nil {
			fmt.Fprintf(stdout, "exists  %s\n", s.path)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(s.path, []byte(s.body), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "created %s\n", s.path)
	}
	return nil
}
