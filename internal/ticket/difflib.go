package ticket

import (
	"fmt"
	"io"
	"slices"
)

type matchBlock struct {
	left  int
	right int
	size  int
}

type opcode struct {
	leftEnd    int
	leftStart  int
	rightEnd   int
	rightStart int
	tag        string
}

// sequenceMatcher is difflib's SequenceMatcher with no junk function, autojunk on.
type sequenceMatcher[T comparable] struct {
	left         []T
	matchBlocks  []matchBlock
	right        []T
	rightIndices map[T][]int
}

func newSequenceMatcher[T comparable](left []T, right []T) *sequenceMatcher[T] {
	indices := map[T][]int{}
	for index, element := range right {
		indices[element] = append(indices[element], index)
	}
	// An element in more than 1% of a long right side is popular, and starts no match.
	if len(right) >= 200 {
		threshold := len(right)/100 + 1
		for element, positions := range indices {
			if len(positions) > threshold {
				delete(indices, element)
			}
		}
	}
	return &sequenceMatcher[T]{left: left, right: right, rightIndices: indices}
}

func (matcher *sequenceMatcher[T]) findLongestMatch(leftLow int, leftHigh int, rightLow int, rightHigh int) matchBlock {
	best := matchBlock{left: leftLow, right: rightLow}
	lengths := map[int]int{}
	for i := leftLow; i < leftHigh; i++ {
		next := map[int]int{}
		for _, j := range matcher.rightIndices[matcher.left[i]] {
			if j < rightLow {
				continue
			}
			if j >= rightHigh {
				break
			}
			length := lengths[j-1] + 1
			next[j] = length
			if length > best.size {
				best = matchBlock{left: i - length + 1, right: j - length + 1, size: length}
			}
		}
		lengths = next
	}
	for best.left > leftLow && best.right > rightLow && matcher.left[best.left-1] == matcher.right[best.right-1] {
		best = matchBlock{left: best.left - 1, right: best.right - 1, size: best.size + 1}
	}
	for best.left+best.size < leftHigh && best.right+best.size < rightHigh && matcher.left[best.left+best.size] == matcher.right[best.right+best.size] {
		best.size++
	}
	return best
}

func (matcher *sequenceMatcher[T]) groupedOpcodes(context int) [][]opcode {
	codes := matcher.opcodes()
	if len(codes) == 0 {
		codes = []opcode{{leftEnd: 1, rightEnd: 1, tag: "equal"}}
	}
	if first := &codes[0]; first.tag == "equal" {
		first.leftStart, first.rightStart = max(first.leftStart, first.leftEnd-context), max(first.rightStart, first.rightEnd-context)
	}
	if last := &codes[len(codes)-1]; last.tag == "equal" {
		last.leftEnd, last.rightEnd = min(last.leftEnd, last.leftStart+context), min(last.rightEnd, last.rightStart+context)
	}
	var groups [][]opcode
	var group []opcode
	for _, code := range codes {
		if code.tag == "equal" && code.leftEnd-code.leftStart > 2*context {
			group = append(group, opcode{leftEnd: min(code.leftEnd, code.leftStart+context), leftStart: code.leftStart, rightEnd: min(code.rightEnd, code.rightStart+context), rightStart: code.rightStart, tag: code.tag})
			groups = append(groups, group)
			group = nil
			code.leftStart, code.rightStart = max(code.leftStart, code.leftEnd-context), max(code.rightStart, code.rightEnd-context)
		}
		group = append(group, code)
	}
	if len(group) > 0 && (len(group) != 1 || group[0].tag != "equal") {
		groups = append(groups, group)
	}
	return groups
}

func (matcher *sequenceMatcher[T]) matchingBlocks() []matchBlock {
	if matcher.matchBlocks != nil {
		return matcher.matchBlocks
	}
	queue := [][4]int{{0, len(matcher.left), 0, len(matcher.right)}}
	var blocks []matchBlock
	for len(queue) > 0 {
		bounds := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		found := matcher.findLongestMatch(bounds[0], bounds[1], bounds[2], bounds[3])
		if found.size == 0 {
			continue
		}
		blocks = append(blocks, found)
		if bounds[0] < found.left && bounds[2] < found.right {
			queue = append(queue, [4]int{bounds[0], found.left, bounds[2], found.right})
		}
		if found.left+found.size < bounds[1] && found.right+found.size < bounds[3] {
			queue = append(queue, [4]int{found.left + found.size, bounds[1], found.right + found.size, bounds[3]})
		}
	}
	slices.SortFunc(blocks, func(first matchBlock, second matchBlock) int {
		if first.left != second.left {
			return first.left - second.left
		}
		if first.right != second.right {
			return first.right - second.right
		}
		return first.size - second.size
	})
	var collapsed []matchBlock
	current := matchBlock{}
	for _, block := range blocks {
		if current.left+current.size == block.left && current.right+current.size == block.right {
			current.size += block.size
			continue
		}
		if current.size > 0 {
			collapsed = append(collapsed, current)
		}
		current = block
	}
	if current.size > 0 {
		collapsed = append(collapsed, current)
	}
	matcher.matchBlocks = append(collapsed, matchBlock{left: len(matcher.left), right: len(matcher.right)})
	return matcher.matchBlocks
}

func (matcher *sequenceMatcher[T]) opcodes() []opcode {
	var codes []opcode
	i, j := 0, 0
	for _, block := range matcher.matchingBlocks() {
		tag := ""
		switch {
		case i < block.left && j < block.right:
			tag = "replace"
		case i < block.left:
			tag = "delete"
		case j < block.right:
			tag = "insert"
		}
		if tag != "" {
			codes = append(codes, opcode{leftEnd: block.left, leftStart: i, rightEnd: block.right, rightStart: j, tag: tag})
		}
		i, j = block.left+block.size, block.right+block.size
		if block.size > 0 {
			codes = append(codes, opcode{leftEnd: i, leftStart: block.left, rightEnd: j, rightStart: block.right, tag: "equal"})
		}
	}
	return codes
}

func (matcher *sequenceMatcher[T]) ratio() float64 {
	matched := 0
	for _, block := range matcher.matchingBlocks() {
		matched += block.size
	}
	if total := len(matcher.left) + len(matcher.right); total > 0 {
		return 2.0 * float64(matched) / float64(total)
	}
	return 1.0
}

func unifiedRange(start int, stop int) string {
	beginning, length := start+1, stop-start
	if length == 1 {
		return fmt.Sprint(beginning)
	}
	if length == 0 {
		beginning--
	}
	return fmt.Sprintf("%d,%d", beginning, length)
}

// Lines keep their endings; a last line without one runs into what follows, as difflib's does.
func writeUnifiedDiff(w io.Writer, left []string, right []string, fromFile string, toFile string) {
	for index, group := range newSequenceMatcher(left, right).groupedOpcodes(3) {
		if index == 0 {
			fmt.Fprintf(w, "--- %s\n+++ %s\n", fromFile, toFile)
		}
		first, last := group[0], group[len(group)-1]
		fmt.Fprintf(w, "@@ -%s +%s @@\n", unifiedRange(first.leftStart, last.leftEnd), unifiedRange(first.rightStart, last.rightEnd))
		for _, code := range group {
			if code.tag == "equal" {
				for _, line := range left[code.leftStart:code.leftEnd] {
					fmt.Fprint(w, " "+line)
				}
				continue
			}
			if code.tag != "insert" {
				for _, line := range left[code.leftStart:code.leftEnd] {
					fmt.Fprint(w, "-"+line)
				}
			}
			if code.tag != "delete" {
				for _, line := range right[code.rightStart:code.rightEnd] {
					fmt.Fprint(w, "+"+line)
				}
			}
		}
	}
}
