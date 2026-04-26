// Package block implements the hatch-managed block-injection protocol used
// for files that may contain user-authored content around a hatch-generated
// section (CLAUDE.md, AGENTS.md, .github/copilot-instructions.md, .rules).
//
// A hatch-managed block is delimited by HTML comments that must appear on
// lines of their own:
//
//	<!-- hatch:begin v1 -->
//	...hatch-generated content...
//	<!-- hatch:end v1 -->
//
// The `v1` token versions the marker so older blocks can be recognized and
// replaced. Everything outside the markers is user-authored and is never
// touched by hatch. Mentions of the marker text inside a paragraph or a
// code fence are not treated as markers — only whole-line matches count.
package block

import (
	"fmt"
	"os"
	"strings"
)

// CurrentMarker is the marker token used by this version of hatch.
const CurrentMarker = "v1"

// Markers returns the begin/end lines for a given marker token.
func Markers(marker string) (begin, end string) {
	return fmt.Sprintf("<!-- hatch:begin %s -->", marker),
		fmt.Sprintf("<!-- hatch:end %s -->", marker)
}

// Render wraps content in begin/end markers and returns the complete block
// text (without surrounding file content).
func Render(marker, content string) string {
	begin, end := Markers(marker)
	content = strings.TrimRight(content, "\n")
	return begin + "\n" + content + "\n" + end + "\n"
}

// Inject writes the hatch block into path. If the file exists and already
// contains a hatch block (for any recognized marker), the block is replaced
// in-place. If the file exists without a block, the block is appended. If
// the file does not exist, it is created with just the block.
//
// Content that embeds the literal begin/end marker text on a line of its
// own is rejected: such content would collide with the block boundaries on
// subsequent rebuilds and corrupt the file. Inline mentions (inside a
// paragraph or a code fence) are fine — hatch only recognises whole-line
// markers.
func Inject(path, marker, content string) error {
	if err := validateContent(content); err != nil {
		return err
	}
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	block := Render(marker, content)
	var out string
	if len(existing) == 0 {
		out = block
	} else {
		out = replaceBlock(string(existing), marker, block)
	}
	if err := os.MkdirAll(parentDir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(out), 0o644)
}

// validateContent refuses content where a line is, verbatim, a hatch
// marker. Inline mentions inside paragraphs or code fences are fine — the
// parser ignores them because it only matches whole lines.
func validateContent(content string) error {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "<!-- hatch:") || !strings.HasSuffix(trimmed, "-->") {
			continue
		}
		kind := trimmed[len("<!-- hatch:"):]
		if strings.HasPrefix(kind, "begin") || strings.HasPrefix(kind, "end") {
			return fmt.Errorf("content contains a bare hatch marker line %q; would corrupt block boundaries", trimmed)
		}
	}
	return nil
}

// Strip removes the hatch block (for the given marker) from path, preserving
// surrounding user content. If the file ends up empty after stripping, it is
// removed entirely.
func Strip(path, marker string) error {
	return stripFile(path, marker)
}

// replaceBlock swaps any existing hatch block in src (matching marker) with
// block. Markers are matched only when they appear on lines of their own.
// If none is found, block is appended with a separating blank line.
func replaceBlock(src, marker, block string) string {
	begin, end := Markers(marker)
	bi := indexMarkerLine(src, 0, begin)
	if bi < 0 {
		// No existing block — append.
		if !strings.HasSuffix(src, "\n") {
			src += "\n"
		}
		if !strings.HasSuffix(src, "\n\n") {
			src += "\n"
		}
		return src + block
	}
	ei := indexMarkerLine(src, bi+len(begin), end)
	if ei < 0 {
		// Malformed (begin line without matching end line); leave the
		// file alone and append.
		return src + "\n" + block
	}
	endLineEnd := ei + len(end)
	for endLineEnd < len(src) && src[endLineEnd] == '\n' {
		endLineEnd++
	}
	tail := src[endLineEnd:]
	head := src[:bi]
	var buf strings.Builder
	buf.WriteString(head)
	buf.WriteString(block)
	if tail != "" {
		if !strings.HasSuffix(block, "\n") {
			buf.WriteString("\n")
		}
		buf.WriteString(tail)
	}
	return buf.String()
}

func stripFile(path, marker string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	begin, end := Markers(marker)
	src := string(data)
	bi := indexMarkerLine(src, 0, begin)
	if bi < 0 {
		return nil
	}
	ei := indexMarkerLine(src, bi+len(begin), end)
	if ei < 0 {
		return fmt.Errorf("begin marker line without matching end marker line")
	}
	endLineEnd := ei + len(end)
	for endLineEnd < len(src) && src[endLineEnd] == '\n' {
		endLineEnd++
	}
	stripped := strings.TrimRight(src[:bi], "\n")
	tail := strings.TrimLeft(src[endLineEnd:], "\n")
	var result string
	switch {
	case stripped == "" && tail == "":
		return os.Remove(path)
	case stripped == "":
		result = tail
	case tail == "":
		result = stripped + "\n"
	default:
		result = stripped + "\n\n" + tail
	}
	return os.WriteFile(path, []byte(result), 0o644)
}

// indexMarkerLine returns the byte offset in src (at or after `from`) where
// a line consisting exactly of marker begins, or -1 if no such line exists.
// A line is "consisting exactly of marker" when the bytes on that line,
// trimmed of surrounding whitespace, equal marker.
func indexMarkerLine(src string, from int, marker string) int {
	offset := from
	// Move to the start of the line containing offset.
	for offset > 0 && src[offset-1] != '\n' {
		offset--
	}
	for offset < len(src) {
		nl := strings.IndexByte(src[offset:], '\n')
		var line string
		var next int
		if nl < 0 {
			line = src[offset:]
			next = len(src)
		} else {
			line = src[offset : offset+nl]
			next = offset + nl + 1
		}
		if strings.TrimSpace(line) == marker && offset >= from {
			return offset
		}
		offset = next
	}
	return -1
}

func parentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
