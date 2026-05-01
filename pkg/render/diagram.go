package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"cli/pkg/api"
)

const branchGap = 3

func WorkflowDiagram(def json.RawMessage) string {
	var wd api.WorkflowDefinition
	if err := json.Unmarshal(def, &wd); err != nil || len(wd.Nodes) == 0 {
		return "  (empty workflow)"
	}

	r := &dagRenderer{
		nodeByID:   make(map[string]*api.WorkflowNode),
		children:   make(map[string][]edgeInfo),
		rendered:   make(map[string]bool),
		widthCache: make(map[string]int),
	}

	inDegree := make(map[string]int)
	for i := range wd.Nodes {
		r.nodeByID[wd.Nodes[i].ID] = &wd.Nodes[i]
		inDegree[wd.Nodes[i].ID] = 0
	}
	for _, e := range wd.Edges {
		r.children[e.Source] = append(r.children[e.Source], edgeInfo{target: e.Target, label: e.Label})
		inDegree[e.Target]++
	}

	r.boxInner = 18
	for _, n := range wd.Nodes {
		content := fmt.Sprintf("%s %s", nodeIcon(n.Type), nodeLabel(&n))
		if w := len(content) + 2; w > r.boxInner {
			r.boxInner = w
		}
	}

	var roots []string
	for id, deg := range inDegree {
		if deg == 0 {
			roots = append(roots, id)
		}
	}

	var allLines []string
	for _, root := range roots {
		allLines = append(allLines, r.renderSubtree(root)...)
	}
	for _, n := range wd.Nodes {
		if !r.rendered[n.ID] {
			allLines = append(allLines, r.renderSubtree(n.ID)...)
		}
	}

	var sb strings.Builder
	for _, line := range allLines {
		sb.WriteString("    " + strings.TrimRight(line, " ") + "\n")
	}
	return sb.String()
}

type edgeInfo struct {
	target string
	label  string
}

type dagRenderer struct {
	nodeByID   map[string]*api.WorkflowNode
	children   map[string][]edgeInfo
	rendered   map[string]bool
	widthCache map[string]int
	boxInner   int
}

func (r *dagRenderer) boxOuter() int { return r.boxInner + 2 }

func (r *dagRenderer) subtreeWidth(id string) int {
	if w, ok := r.widthCache[id]; ok {
		return w
	}
	w := r.boxOuter()
	edges := r.children[id]
	if len(edges) == 1 {
		if cw := r.subtreeWidth(edges[0].target); cw > w {
			w = cw
		}
	} else if len(edges) > 1 {
		total := 0
		for i, e := range edges {
			total += r.subtreeWidth(e.target)
			if i < len(edges)-1 {
				total += branchGap
			}
		}
		if total > w {
			w = total
		}
	}
	r.widthCache[id] = w
	return w
}

// renderSubtree returns lines all padded to subtreeWidth(id).
func (r *dagRenderer) renderSubtree(id string) []string {
	if r.rendered[id] {
		w := r.subtreeWidth(id)
		node := r.nodeByID[id]
		if node == nil {
			return nil
		}
		ref := "(-> " + nodeLabel(node) + ")"
		return []string{centerIn(ref, w)}
	}
	r.rendered[id] = true

	node := r.nodeByID[id]
	if node == nil {
		return nil
	}

	w := r.subtreeWidth(id)
	bw := r.boxOuter()

	result := centerBlock(r.makeBox(node), bw, w)

	edges := r.children[id]
	if len(edges) == 0 {
		return result
	}

	center := w / 2

	if len(edges) == 1 {
		e := edges[0]
		result = append(result, lineWith(w, center, '|'))
		if e.label != "" {
			result = append(result, centerIn("["+e.label+"]", w))
			result = append(result, lineWith(w, center, '|'))
		}
		result = append(result, lineWith(w, center, 'v'))
		childW := r.subtreeWidth(e.target)
		result = append(result, centerBlock(r.renderSubtree(e.target), childW, w)...)
		return result
	}

	// --- Fork: multiple children ---
	childWidths := make([]int, len(edges))
	childTrees := make([][]string, len(edges))
	for i, e := range edges {
		childWidths[i] = r.subtreeWidth(e.target)
		childTrees[i] = r.renderSubtree(e.target)
	}

	childrenTotal := 0
	for i, cw := range childWidths {
		childrenTotal += cw
		if i < len(childWidths)-1 {
			childrenTotal += branchGap
		}
	}

	offset := (w - childrenTotal) / 2
	if offset < 0 {
		offset = 0
	}

	childCenters := make([]int, len(edges))
	x := offset
	for i, cw := range childWidths {
		childCenters[i] = x + cw/2
		x += cw + branchGap
	}

	// Vertical pipe from parent center
	result = append(result, lineWith(w, center, '|'))

	// Horizontal line connecting all child centers
	leftC := childCenters[0]
	rightC := childCenters[len(childCenters)-1]
	hLine := make([]byte, w)
	for i := range hLine {
		hLine[i] = ' '
	}
	for i := leftC; i <= rightC; i++ {
		hLine[i] = '-'
	}
	if center >= 0 && center < w {
		hLine[center] = '+'
	}
	for _, c := range childCenters {
		if c >= 0 && c < w {
			hLine[c] = '+'
		}
	}
	result = append(result, string(hLine))

	// Edge labels (if any exist)
	hasLabels := false
	for _, e := range edges {
		if e.label != "" {
			hasLabels = true
			break
		}
	}
	if hasLabels {
		lbl := make([]byte, w)
		for i := range lbl {
			lbl[i] = ' '
		}
		for i, e := range edges {
			if e.label != "" {
				tag := "[" + e.label + "]"
				start := childCenters[i] - len(tag)/2
				for j := 0; j < len(tag); j++ {
					p := start + j
					if p >= 0 && p < w {
						lbl[p] = tag[j]
					}
				}
			}
		}
		result = append(result, string(lbl))
	}

	// Vertical pipes + arrows at each child center
	pipes := make([]byte, w)
	arrows := make([]byte, w)
	for i := range pipes {
		pipes[i] = ' '
		arrows[i] = ' '
	}
	for _, c := range childCenters {
		if c >= 0 && c < w {
			pipes[c] = '|'
			arrows[c] = 'v'
		}
	}
	result = append(result, string(pipes))
	result = append(result, string(arrows))

	// Combine child subtrees side by side
	combined := placeSideBySide(childTrees, childWidths, branchGap)
	for _, line := range combined {
		padded := strings.Repeat(" ", offset) + line
		if len(padded) < w {
			padded += strings.Repeat(" ", w-len(padded))
		}
		result = append(result, padded)
	}

	return result
}

func (r *dagRenderer) makeBox(node *api.WorkflowNode) []string {
	content := fmt.Sprintf("%s %s", nodeIcon(node.Type), nodeLabel(node))
	bw := r.boxInner
	pad := bw - 2 - len(content)
	if pad < 0 {
		pad = 0
	}
	border := strings.Repeat("-", bw)
	return []string{
		"+" + border + "+",
		"| " + content + strings.Repeat(" ", pad) + " |",
		"+" + border + "+",
	}
}

// --- helpers ---

func nodeLabel(n *api.WorkflowNode) string {
	var data api.NodeData
	if err := json.Unmarshal(n.Data, &data); err == nil && data.Label != "" {
		return data.Label
	}
	return n.ID
}

func nodeIcon(nodeType string) string {
	switch nodeType {
	case "trigger":
		return ">>"
	case "action":
		return "*"
	case "condition":
		return "?"
	case "approval":
		return "!"
	default:
		return "-"
	}
}

func lineWith(width, pos int, ch byte) string {
	b := make([]byte, width)
	for i := range b {
		b[i] = ' '
	}
	if pos >= 0 && pos < width {
		b[pos] = ch
	}
	return string(b)
}

func centerIn(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	pad := (width - len(s)) / 2
	return strings.Repeat(" ", pad) + s + strings.Repeat(" ", width-pad-len(s))
}

func centerBlock(lines []string, blockW, totalW int) []string {
	if blockW >= totalW {
		return lines
	}
	pad := (totalW - blockW) / 2
	result := make([]string, len(lines))
	for i, line := range lines {
		right := totalW - pad - len(line)
		if right < 0 {
			right = 0
		}
		result[i] = strings.Repeat(" ", pad) + line + strings.Repeat(" ", right)
	}
	return result
}

func placeSideBySide(groups [][]string, widths []int, gap int) []string {
	maxH := 0
	for _, g := range groups {
		if len(g) > maxH {
			maxH = len(g)
		}
	}
	gapStr := strings.Repeat(" ", gap)
	result := make([]string, maxH)
	for row := 0; row < maxH; row++ {
		var parts []string
		for i, g := range groups {
			if row < len(g) {
				line := g[row]
				if len(line) < widths[i] {
					line += strings.Repeat(" ", widths[i]-len(line))
				}
				parts = append(parts, line)
			} else {
				parts = append(parts, strings.Repeat(" ", widths[i]))
			}
		}
		result[row] = strings.Join(parts, gapStr)
	}
	return result
}
