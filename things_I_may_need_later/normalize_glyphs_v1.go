package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Command represents one SVG path command with its arguments.
type Command struct {
	Cmd  rune
	Args []float64
}

// ParsePath tokenizes an SVG path d string into commands.
// It handles the common cases in the character-definitions file (M L Q A Z, space-separated numbers, A flags).
func ParsePath(d string) []Command {
	d = strings.TrimSpace(d)
	if d == "" {
		return nil
	}

	// Insert spaces around command letters so we can split cleanly
	re := regexp.MustCompile(`([A-Za-z])`)
	d = re.ReplaceAllString(d, " $1 ")

	tokens := strings.Fields(d)
	var cmds []Command
	var current Command
	var args []float64

	for _, tok := range tokens {
		if len(tok) == 1 && (tok[0] >= 'A' && tok[0] <= 'Z' || tok[0] >= 'a' && tok[0] <= 'z') {
			if current.Cmd != 0 {
				current.Args = args
				cmds = append(cmds, current)
			}
			current = Command{Cmd: rune(tok[0])}
			args = nil
			continue
		}
		// number (including possible leading sign, decimal, etc.)
		f, err := strconv.ParseFloat(tok, 64)
		if err == nil {
			args = append(args, f)
		}
	}
	if current.Cmd != 0 {
		current.Args = args
		cmds = append(cmds, current)
	}
	return cmds
}

// bake adds tx to every X coordinate and ty to every Y coordinate in the commands.
// For 'A' arcs we only shift the final endpoint (args[5], args[6] in absolute form).
func bake(cmds []Command, tx, ty float64) []Command {
	out := make([]Command, len(cmds))
	for i, c := range cmds {
		out[i].Cmd = c.Cmd
		out[i].Args = make([]float64, len(c.Args))
		copy(out[i].Args, c.Args)

		switch c.Cmd {
		case 'M', 'L', 'Q', 'T', 'C', 'S': // pairs of coords
			for j := 0; j < len(out[i].Args); j += 2 {
				out[i].Args[j] += tx
				if j+1 < len(out[i].Args) {
					out[i].Args[j+1] += ty
				}
			}
		case 'H':
			for j := range out[i].Args {
				out[i].Args[j] += tx
			}
		case 'V':
			for j := range out[i].Args {
				out[i].Args[j] += ty
			}
		case 'A':
			// A rx ry xrot large sweep x y  → only shift final x y (indices 5,6)
			if len(out[i].Args) >= 7 {
				out[i].Args[5] += tx
				out[i].Args[6] += ty
			}
		}
	}
	return out
}

// toRelative converts baked absolute commands to relative form.
// It tracks current point and subpath start for Z.
func toRelative(cmds []Command) []Command {
	if len(cmds) == 0 {
		return nil
	}

	var out []Command
	var curX, curY float64
	var subStartX, subStartY float64
	firstMove := true

	for _, c := range cmds {
		newCmd := Command{Cmd: toLower(c.Cmd)}
		newArgs := make([]float64, len(c.Args))
		copy(newArgs, c.Args)

		switch c.Cmd {
		case 'M':
			if firstMove {
				// First M becomes m dx dy from implicit (0,0) of the cell
				newCmd.Cmd = 'm'
				// newArgs already has the baked absolute values → they are the deltas from (0,0)
				curX, curY = newArgs[0], newArgs[1]
				subStartX, subStartY = curX, curY
				firstMove = false
			} else {
				// Subsequent M (new subpath) is relative to current
				newArgs[0] -= curX
				newArgs[1] -= curY
				curX += newArgs[0]
				curY += newArgs[1]
				subStartX, subStartY = curX, curY
			}
		case 'L':
			newArgs[0] -= curX
			newArgs[1] -= curY
			curX += newArgs[0]
			curY += newArgs[1]
		case 'Q':
			newArgs[0] -= curX // cx
			newArgs[1] -= curY // cy
			newArgs[2] -= curX // x
			newArgs[3] -= curY // y
			curX += newArgs[2]
			curY += newArgs[3]
		case 'A':
			if len(newArgs) >= 7 {
				newArgs[5] -= curX
				newArgs[6] -= curY
				curX += newArgs[5]
				curY += newArgs[6]
			}
		case 'Z', 'z':
			curX, curY = subStartX, subStartY
			newCmd.Cmd = 'z'
			newArgs = nil
		}

		newCmd.Args = newArgs
		out = append(out, newCmd)
	}

	// Append the advance to next cell top-left
	out = append(out, Command{Cmd: 'm', Args: []float64{8, 0}})

	return out
}

func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}

// SerializePath turns commands back into a d string. Rounds to 4 decimals for cleanliness.
func SerializePath(cmds []Command) string {
	var b strings.Builder
	for i, c := range cmds {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(c.Cmd)
		for j, a := range c.Args {
			if j > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(strconv.FormatFloat(a, 'f', 4, 64))
		}
	}
	return b.String()
}

// ProcessGlyphPath takes one path's original d and its group's tx,ty and returns the final relative + advance d string.
func ProcessGlyphPath(originalD string, tx, ty float64) string {
	cmds := ParsePath(originalD)
	baked := bake(cmds, tx, ty)
	rel := toRelative(baked)
	return SerializePath(rel)
}

func main() {
	// Demo on the apostrophe from your file
	apostropheD := `M 0 0 L 1.408 0 Q 1.408 0.96 1.392 1.536 Q 1.376 2.112 1.328 2.504 Q 1.28 2.896 1.232 3.256 Q 1.184 3.616 1.136 4.128 L 0.272 4.128 Q 0.224 3.616 0.176 3.256 Q 0.128 2.896 0.08 2.512 Q 0.032 2.128 0.016 1.544 Q 0.002 1.022 0 0.198 A 108.186 108.186 0 0 1 0 0 Z`
	tx, ty := 3.30, 2.35

	result := ProcessGlyphPath(apostropheD, tx, ty)
	fmt.Println("Apostrophe relative + advance:")
	fmt.Println(result)
	fmt.Println()

	// You can extend main() to read the full SVG file, find every g + its transform + paths,
	// call ProcessGlyphPath for each, and rebuild a new SVG or emit a Go map[rune]string.
	fmt.Println("To process the whole file, extend this program with xml parsing + loop over g elements.")
}