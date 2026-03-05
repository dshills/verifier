package parse

import (
	"strings"
)

// Heading represents a parsed markdown heading.
type Heading struct {
	Level int
	Text  string
	Line  int
}

// Section represents a heading and its body content.
type Section struct {
	Heading Heading
	Body    []string // lines between this heading and the next
}

// ParseSections splits markdown into heading-delimited sections.
func ParseSections(text string) []Section {
	lines := strings.Split(text, "\n")
	var sections []Section
	var current *Section

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check ATX headings (# Heading)
		if level := atxLevel(line); level > 0 {
			if current != nil {
				sections = append(sections, *current)
			}
			current = &Section{
				Heading: Heading{
					Level: level,
					Text:  strings.TrimSpace(strings.TrimLeft(line, "# ")),
					Line:  i + 1,
				},
			}
			continue
		}

		// Check setext headings (underline with = or -)
		if i+1 < len(lines) && current != nil {
			nextLine := lines[i+1]
			if isSetextUnderline(nextLine) {
				sections = append(sections, *current)
				level := 1
				if strings.HasPrefix(nextLine, "-") {
					level = 2
				}
				current = &Section{
					Heading: Heading{
						Level: level,
						Text:  strings.TrimSpace(line),
						Line:  i + 1,
					},
				}
				i++ // skip underline
				continue
			}
		}

		if current != nil {
			current.Body = append(current.Body, line)
		}
	}

	if current != nil {
		sections = append(sections, *current)
	}

	return sections
}

func atxLevel(line string) int {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return 0
	}
	level := 0
	for _, ch := range trimmed {
		if ch == '#' {
			level++
		} else {
			break
		}
	}
	if level > 6 || level == 0 {
		return 0
	}
	// Must have a space after # (or be empty heading)
	rest := trimmed[level:]
	if len(rest) > 0 && rest[0] != ' ' {
		return 0
	}
	return level
}

func isSetextUnderline(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 2 {
		return false
	}
	allEq := true
	allDash := true
	for _, ch := range trimmed {
		if ch != '=' {
			allEq = false
		}
		if ch != '-' {
			allDash = false
		}
	}
	return allEq || allDash
}

// ExtractListItems extracts bulleted/numbered list items from body lines.
func ExtractListItems(lines []string) []string {
	var items []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if item, ok := parseListItem(trimmed); ok {
			items = append(items, item)
		}
	}
	return items
}

func parseListItem(line string) (string, bool) {
	// Bulleted: - item or * item
	if strings.HasPrefix(line, "- ") {
		return strings.TrimSpace(line[2:]), true
	}
	if strings.HasPrefix(line, "* ") {
		return strings.TrimSpace(line[2:]), true
	}
	// Numbered: 1. item or 1) item
	for i, ch := range line {
		if ch >= '0' && ch <= '9' {
			continue
		}
		if (ch == '.' || ch == ')') && i > 0 && i+1 < len(line) && line[i+1] == ' ' {
			return strings.TrimSpace(line[i+2:]), true
		}
		break
	}
	return "", false
}
