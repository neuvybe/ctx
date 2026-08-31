package ctx

import "strings"

const (
	// managedBegin marks the start of a platform-owned block. ctx update rewrites
	// the content between begin and end from its embedded templates; everything
	// outside these markers is user-owned and preserved verbatim.
	managedBegin = "<!-- ctx:managed begin -->"
	managedEnd   = "<!-- ctx:managed end -->"
)

// managedBlock holds the inner lines of one managed block (markers excluded).
type managedBlock struct {
	innerLines []string
}

// parseManaged returns the managed blocks in content, in order.
func parseManaged(content string) []managedBlock {
	var blocks []managedBlock
	var cur []string
	inBlock := false
	for _, line := range strings.Split(content, "\n") {
		switch strings.TrimSpace(line) {
		case managedBegin:
			inBlock = true
			cur = nil
		case managedEnd:
			if inBlock {
				blocks = append(blocks, managedBlock{innerLines: cur})
			}
			inBlock = false
			cur = nil
		default:
			if inBlock {
				cur = append(cur, line)
			}
		}
	}
	return blocks
}

// hasManaged reports whether content contains any managed block.
func hasManaged(content string) bool {
	return len(parseManaged(content)) > 0
}

// markersBalanced reports whether markers form a strict sequence of complete,
// non-nested begin/end pairs. Markerless content is valid and user-owned.
func markersBalanced(content string) bool {
	inBlock := false
	for _, line := range strings.Split(content, "\n") {
		switch strings.TrimSpace(line) {
		case managedBegin:
			if inBlock {
				return false
			}
			inBlock = true
		case managedEnd:
			if !inBlock {
				return false
			}
			inBlock = false
		}
	}
	return !inBlock
}

// updateManagedContent returns existing with each managed block's inner content
// replaced by the corresponding (by ordinal) managed block from newTemplate.
// User text (everything outside managed blocks) is preserved verbatim from
// existing. If existing has more managed blocks than newTemplate, the extras are
// left as-is (not refreshed). If existing has fewer, the extra new blocks are
// dropped (they have no anchor in existing to attach to — call it a re-init).
// Returns the result plus the managed-block counts for mismatch reporting.
func updateManagedContent(existing, newTemplate string) (string, int, int) {
	newBlocks := parseManaged(newTemplate)
	existingCount := len(parseManaged(existing))

	var out []string
	inBlock := false
	idx := -1
	for _, line := range strings.Split(existing, "\n") {
		t := strings.TrimSpace(line)
		if t == managedBegin {
			inBlock = true
			idx++
			out = append(out, line) // keep existing begin marker
			if idx < len(newBlocks) {
				out = append(out, newBlocks[idx].innerLines...) // swap in new inner content
			}
			continue
		}
		if t == managedEnd {
			inBlock = false
			out = append(out, line) // keep existing end marker
			continue
		}
		if inBlock {
			// Preserve existing inner lines only when there's no new block to swap in.
			if idx >= len(newBlocks) {
				out = append(out, line)
			}
			continue
		}
		out = append(out, line) // user text, preserve verbatim
	}
	return strings.Join(out, "\n"), existingCount, len(newBlocks)
}
