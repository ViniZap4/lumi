package ui

import (
	"strings"
	"unicode/utf8"
)

// tableBlock describes a contiguous range of table lines in the content.
type tableBlock struct {
	start int // first line index (inclusive)
	end   int // last line index (inclusive)
}

// tableLineCtx marks the role of a line within a table block.
type tableLineCtx struct {
	isHeader    bool
	isSeparator bool
	isData      bool
}

// preprocessTableBlocks finds table blocks in the content lines and pads their
// cells so columns are aligned. It modifies the slice in place.
func preprocessTableBlocks(lines []string) {
	blocks := findTableBlocks(lines)
	for _, blk := range blocks {
		alignAndPadTable(lines, blk)
	}
}

// findTableBlocks scans lines for consecutive pipe-delimited table rows,
// skipping anything inside fenced code blocks.
func findTableBlocks(lines []string) []tableBlock {
	var blocks []tableBlock
	inCode := false
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
			i++
			continue
		}
		if inCode {
			i++
			continue
		}
		if isTableLine(trimmed) {
			start := i
			for i < len(lines) {
				t := strings.TrimSpace(lines[i])
				if !isTableLine(t) {
					break
				}
				i++
			}
			// A table needs at least 2 lines (header + separator)
			if i-start >= 2 {
				blocks = append(blocks, tableBlock{start: start, end: i - 1})
			}
		} else {
			i++
		}
	}
	return blocks
}

// alignAndPadTable parses cells in a table block, computes maximum column widths,
// and rewrites each line with padded cells. Alignment markers on the separator
// row (`:---`, `---:`, `:---:`) determine left/right/center padding.
func alignAndPadTable(lines []string, blk tableBlock) {
	type cellInfo struct {
		cells []string
	}

	rows := make([]cellInfo, 0, blk.end-blk.start+1)
	maxCols := 0

	// Parse cells from each row
	for i := blk.start; i <= blk.end; i++ {
		trimmed := strings.TrimSpace(lines[i])
		// Remove leading/trailing pipes
		inner := trimmed[1 : len(trimmed)-1]
		parts := strings.Split(inner, "|")
		cells := make([]string, len(parts))
		for j, p := range parts {
			cells[j] = strings.TrimSpace(p)
		}
		rows = append(rows, cellInfo{cells: cells})
		if len(cells) > maxCols {
			maxCols = len(cells)
		}
	}

	if maxCols == 0 || len(rows) == 0 {
		return
	}

	// Compute max width per column
	colWidths := make([]int, maxCols)
	for _, row := range rows {
		for j, cell := range row.cells {
			w := utf8.RuneCountInString(cell)
			if w > colWidths[j] {
				colWidths[j] = w
			}
		}
	}

	// Enforce minimum column width of 3 (for separator dashes)
	for j := range colWidths {
		if colWidths[j] < 3 {
			colWidths[j] = 3
		}
	}

	// Detect alignment from separator row (second row if it exists)
	type alignment int
	const (
		alignLeft alignment = iota
		alignRight
		alignCenter
	)
	aligns := make([]alignment, maxCols)
	if len(rows) >= 2 {
		sepRow := rows[1]
		for j, cell := range sepRow.cells {
			cell = strings.TrimSpace(cell)
			left := strings.HasPrefix(cell, ":")
			right := strings.HasSuffix(cell, ":")
			switch {
			case left && right:
				aligns[j] = alignCenter
			case right:
				aligns[j] = alignRight
			default:
				aligns[j] = alignLeft
			}
		}
	}

	// Rebuild each line
	for ri, row := range rows {
		lineIdx := blk.start + ri
		var sb strings.Builder
		sb.WriteString("|")
		isSep := ri == 1 // separator row
		for j := 0; j < maxCols; j++ {
			cell := ""
			if j < len(row.cells) {
				cell = row.cells[j]
			}
			w := colWidths[j]
			sb.WriteString(" ")
			if isSep {
				// Rebuild separator cell with alignment markers
				switch aligns[j] {
				case alignCenter:
					sb.WriteString(":")
					sb.WriteString(strings.Repeat("-", w-2))
					sb.WriteString(":")
				case alignRight:
					sb.WriteString(strings.Repeat("-", w-1))
					sb.WriteString(":")
				default:
					sb.WriteString(strings.Repeat("-", w))
				}
			} else {
				cellWidth := utf8.RuneCountInString(cell)
				pad := w - cellWidth
				switch aligns[j] {
				case alignCenter:
					left := pad / 2
					right := pad - left
					sb.WriteString(strings.Repeat(" ", left))
					sb.WriteString(cell)
					sb.WriteString(strings.Repeat(" ", right))
				case alignRight:
					sb.WriteString(strings.Repeat(" ", pad))
					sb.WriteString(cell)
				default:
					sb.WriteString(cell)
					sb.WriteString(strings.Repeat(" ", pad))
				}
			}
			sb.WriteString(" |")
		}
		lines[lineIdx] = sb.String()
	}
}

// buildTableLineCtx creates a map marking each line's role in a table block.
func buildTableLineCtx(lines []string) map[int]tableLineCtx {
	ctx := map[int]tableLineCtx{}
	blocks := findTableBlocks(lines)
	for _, blk := range blocks {
		for i := blk.start; i <= blk.end; i++ {
			switch {
			case i == blk.start:
				ctx[i] = tableLineCtx{isHeader: true}
			case i == blk.start+1:
				ctx[i] = tableLineCtx{isSeparator: true}
			default:
				ctx[i] = tableLineCtx{isData: true}
			}
		}
	}
	return ctx
}
