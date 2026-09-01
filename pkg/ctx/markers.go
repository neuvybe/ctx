package ctx

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	// managedBegin and managedEnd are the unnamed markers emitted by v1
	// layouts. They remain supported only for v1 compatibility.
	managedBegin = "<!-- ctx:managed begin -->"
	managedEnd   = "<!-- ctx:managed end -->"
)

type managedMarkerFormat uint8

const (
	managedMarkersUnnamed managedMarkerFormat = iota + 1
	managedMarkersNamed
)

func managedMarkerFormatFor(format MarkerFormat) (managedMarkerFormat, error) {
	switch format {
	case MarkerFormatUnnamedV1:
		return managedMarkersUnnamed, nil
	case MarkerFormatNamedV2:
		return managedMarkersNamed, nil
	default:
		return 0, fmt.Errorf("unsupported managed-marker format %q", format)
	}
}

var managedIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

// managedBlock holds one parsed managed block. The offsets delimit only the
// block body, so replacing that range preserves both marker lines and every
// byte outside the block.
type managedBlock struct {
	id         string
	innerLines []string
	body       string
	bodyStart  int
	bodyEnd    int
}

type managedDocument struct {
	blocks []managedBlock
}

type contentLine struct {
	text       string
	start, end int
}

// parseManaged returns the legacy unnamed managed blocks in content, in order.
// Callers that need validation use parseManagedDocument so malformed input is
// not mistaken for markerless, user-owned content.
func parseManaged(content string) []managedBlock {
	doc, err := parseManagedDocument(content, managedMarkersUnnamed)
	if err != nil {
		return nil
	}
	return doc.blocks
}

// hasManaged reports whether content contains a valid legacy managed block.
func hasManaged(content string) bool {
	return hasManagedForFormat(content, managedMarkersUnnamed)
}

func hasManagedForFormat(content string, format managedMarkerFormat) bool {
	doc, err := parseManagedDocument(content, format)
	return err == nil && len(doc.blocks) > 0
}

// markersBalanced reports whether markers form a valid v1 sequence. Markerless
// content is valid and user-owned. Named markers are deliberately invalid here:
// only v2 layouts may use them.
func markersBalanced(content string) bool {
	return markersValidForFormat(content, managedMarkersUnnamed)
}

func markersValidForFormat(content string, format managedMarkerFormat) bool {
	_, err := parseManagedDocument(content, format)
	return err == nil
}

// parseManagedDocument validates and parses the marker grammar for one layout.
// V1 permits only unnamed ordinal blocks (with its historical surrounding-space
// tolerance). V2 markers must occupy an exact line, carry a lowercase kebab ID,
// be unique within the file, and have a matching end ID.
func parseManagedDocument(content string, format managedMarkerFormat) (managedDocument, error) {
	if format != managedMarkersUnnamed && format != managedMarkersNamed {
		return managedDocument{}, fmt.Errorf("unsupported managed-marker format %d", format)
	}

	lines := contentLines(content)
	var doc managedDocument
	var current *managedBlock
	seen := make(map[string]bool)
	for i, line := range lines {
		kind, id, markerLike, err := classifyManagedMarker(line.text, format)
		if err != nil {
			return managedDocument{}, fmt.Errorf("line %d: %w", i+1, err)
		}
		if kind == "" {
			if markerLike {
				return managedDocument{}, fmt.Errorf("line %d: marker does not match the layout's managed-marker grammar", i+1)
			}
			continue
		}

		switch kind {
		case "begin":
			if current != nil {
				return managedDocument{}, fmt.Errorf("line %d: nested managed block", i+1)
			}
			if format == managedMarkersNamed && seen[id] {
				return managedDocument{}, fmt.Errorf("line %d: duplicate managed block ID %q", i+1, id)
			}
			current = &managedBlock{id: id, bodyStart: line.end}
		case "end":
			if current == nil {
				return managedDocument{}, fmt.Errorf("line %d: managed end marker has no begin marker", i+1)
			}
			if format == managedMarkersNamed && current.id != id {
				return managedDocument{}, fmt.Errorf("line %d: managed end ID %q does not match begin ID %q", i+1, id, current.id)
			}
			current.bodyEnd = line.start
			current.body = content[current.bodyStart:current.bodyEnd]
			current.innerLines = managedInnerLines(current.body)
			doc.blocks = append(doc.blocks, *current)
			if format == managedMarkersNamed {
				seen[id] = true
			}
			current = nil
		}
	}
	if current != nil {
		if current.id == "" {
			return managedDocument{}, fmt.Errorf("managed begin marker has no end marker")
		}
		return managedDocument{}, fmt.Errorf("managed begin marker %q has no end marker", current.id)
	}
	return doc, nil
}

// updateManagedContent returns the v1-compatible ordinal rewrite result. It is
// kept for compatibility with existing tests and helpers; Update uses the
// strict variant below so a mismatch cannot be followed by a version stamp.
func updateManagedContent(existing, newTemplate string) (string, int, int) {
	existingDoc, existingErr := parseManagedDocument(existing, managedMarkersUnnamed)
	templateDoc, templateErr := parseManagedDocument(newTemplate, managedMarkersUnnamed)
	if existingErr != nil || templateErr != nil {
		return existing, len(existingDoc.blocks), len(templateDoc.blocks)
	}

	replacements := make(map[int]string)
	for i := range existingDoc.blocks {
		if i < len(templateDoc.blocks) {
			replacements[i] = templateDoc.blocks[i].body
		}
	}
	return replaceManagedBodies(existing, existingDoc.blocks, replacements), len(existingDoc.blocks), len(templateDoc.blocks)
}

// updateManagedContentStrict refreshes every managed block or returns an error.
// V1 maps blocks by ordinal but requires equal counts. V2 maps by stable ID and
// requires the target and template to contain exactly the same unique ID set.
func updateManagedContentStrict(existing, newTemplate string, format managedMarkerFormat) (string, error) {
	existingDoc, err := parseManagedDocument(existing, format)
	if err != nil {
		return "", fmt.Errorf("target: %w", err)
	}
	templateDoc, err := parseManagedDocument(newTemplate, format)
	if err != nil {
		return "", fmt.Errorf("template: %w", err)
	}
	replacements, err := managedReplacementMap(existingDoc, templateDoc, format)
	if err != nil {
		return "", err
	}
	return replaceManagedBodies(existing, existingDoc.blocks, replacements), nil
}

// validateManagedCompatibility checks the same grammar and block mapping that
// Update will use without producing rewritten content. Doctor uses it to catch
// a structurally stale managed file before an update is attempted.
func validateManagedCompatibility(existing, newTemplate string, format managedMarkerFormat) error {
	existingDoc, err := parseManagedDocument(existing, format)
	if err != nil {
		return fmt.Errorf("target: %w", err)
	}
	templateDoc, err := parseManagedDocument(newTemplate, format)
	if err != nil {
		return fmt.Errorf("template: %w", err)
	}
	_, err = managedReplacementMap(existingDoc, templateDoc, format)
	return err
}

func managedReplacementMap(existingDoc, templateDoc managedDocument, format managedMarkerFormat) (map[int]string, error) {
	// V1 deliberately treats a markerless target as user-owned. V2's named
	// grammar is a structural contract, so an empty target must still be checked
	// against the template's ID set and will fail when IDs are missing.
	if format == managedMarkersUnnamed && len(existingDoc.blocks) == 0 {
		return map[int]string{}, nil
	}
	if len(templateDoc.blocks) == 0 {
		if len(existingDoc.blocks) == 0 {
			return map[int]string{}, nil
		}
		return nil, fmt.Errorf("target has managed blocks but template has none")
	}

	replacements := make(map[int]string, len(existingDoc.blocks))
	switch format {
	case managedMarkersUnnamed:
		if len(existingDoc.blocks) != len(templateDoc.blocks) {
			return nil, fmt.Errorf("managed block count mismatch: target has %d, template has %d", len(existingDoc.blocks), len(templateDoc.blocks))
		}
		for i := range existingDoc.blocks {
			replacements[i] = templateDoc.blocks[i].body
		}
	case managedMarkersNamed:
		templateByID := make(map[string]managedBlock, len(templateDoc.blocks))
		for _, block := range templateDoc.blocks {
			templateByID[block.id] = block
		}
		var missing, extra []string
		targetIDs := make(map[string]bool, len(existingDoc.blocks))
		for i, block := range existingDoc.blocks {
			targetIDs[block.id] = true
			templateBlock, ok := templateByID[block.id]
			if !ok {
				missing = append(missing, block.id)
				continue
			}
			replacements[i] = templateBlock.body
		}
		for id := range templateByID {
			if !targetIDs[id] {
				extra = append(extra, id)
			}
		}
		if len(missing) > 0 || len(extra) > 0 {
			sort.Strings(missing)
			sort.Strings(extra)
			return nil, fmt.Errorf("managed block ID mismatch: target-only=%v template-only=%v", missing, extra)
		}
	default:
		return nil, fmt.Errorf("unsupported managed-marker format %d", format)
	}
	return replacements, nil
}

func replaceManagedBodies(content string, blocks []managedBlock, replacements map[int]string) string {
	var out strings.Builder
	out.Grow(len(content))
	cursor := 0
	for i, block := range blocks {
		out.WriteString(content[cursor:block.bodyStart])
		if replacement, ok := replacements[i]; ok {
			out.WriteString(replacement)
		} else {
			out.WriteString(content[block.bodyStart:block.bodyEnd])
		}
		cursor = block.bodyEnd
	}
	out.WriteString(content[cursor:])
	return out.String()
}

func classifyManagedMarker(line string, format managedMarkerFormat) (kind, id string, markerLike bool, err error) {
	trimmed := strings.TrimSpace(line)
	markerLike = strings.HasPrefix(trimmed, "<!-- ctx:managed")
	if format == managedMarkersUnnamed {
		switch trimmed {
		case managedBegin:
			return "begin", "", true, nil
		case managedEnd:
			return "end", "", true, nil
		default:
			return "", "", markerLike, nil
		}
	}

	// Named markers intentionally do not trim horizontal whitespace. This makes
	// their syntax deterministic while still accepting CRLF as a line ending.
	if line == managedBegin || line == managedEnd {
		return "", "", true, nil
	}
	for _, candidate := range []struct {
		prefix string
		kind   string
	}{
		{prefix: "<!-- ctx:managed begin ", kind: "begin"},
		{prefix: "<!-- ctx:managed end ", kind: "end"},
	} {
		if strings.HasPrefix(line, candidate.prefix) && strings.HasSuffix(line, " -->") {
			id = strings.TrimSuffix(strings.TrimPrefix(line, candidate.prefix), " -->")
			if !managedIDPattern.MatchString(id) {
				return "", "", true, fmt.Errorf("invalid managed block ID %q (want lowercase kebab-case)", id)
			}
			return candidate.kind, id, true, nil
		}
	}
	return "", "", markerLike, nil
}

func contentLines(content string) []contentLine {
	if content == "" {
		return nil
	}
	lines := make([]contentLine, 0, strings.Count(content, "\n")+1)
	start := 0
	for start < len(content) {
		relEnd := strings.IndexByte(content[start:], '\n')
		end := len(content)
		textEnd := end
		if relEnd >= 0 {
			textEnd = start + relEnd
			end = textEnd + 1
		}
		if textEnd > start && content[textEnd-1] == '\r' {
			textEnd--
		}
		lines = append(lines, contentLine{text: content[start:textEnd], start: start, end: end})
		start = end
	}
	return lines
}

func managedInnerLines(body string) []string {
	body = strings.TrimSuffix(body, "\n")
	body = strings.TrimSuffix(body, "\r")
	if body == "" {
		return nil
	}
	lines := strings.Split(body, "\n")
	for i := range lines {
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}
	return lines
}
