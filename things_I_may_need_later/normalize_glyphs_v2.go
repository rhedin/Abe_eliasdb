package main

import (
	"fmt"
	"os"
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
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run normalize_glyphs.go input.html output.html")
		fmt.Println("Example: go run normalize_glyphs.go /path/to/character-definitions.html character-definitions-relative.html")
		return
	}
	inputFile := os.Args[1]
	outputFile := os.Args[2]

	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input %s: %v\n", inputFile, err)
		os.Exit(1)
	}
	content := string(data)

	// Regex to capture each ascii g block with its transform translate(tx,ty)
	re := regexp.MustCompile(`(?s)<g\s+id="(ascii[0-9a-f]+)"([^>]*)transform="translate\(([0-9.-]+),([0-9.-]+)\)"([^>]*)>(.*?)</g>`)

	newContent := re.ReplaceAllStringFunc(content, func(match string) string {
		subs := re.FindStringSubmatch(match)
		if len(subs) != 7 {
			return match
		}
		id := subs[1]
		beforeTx := subs[2]
		txStr := subs[3]
		tyStr := subs[4]
		afterTx := subs[5]
		inner := subs[6]

		tx, _ := strconv.ParseFloat(txStr, 64)
		ty, _ := strconv.ParseFloat(tyStr, 64)

		// Process every <path d="..."> inside this g
		pathRe := regexp.MustCompile(`<path([^>]*)\sd="([^"]*)"([^>]*)>`)
		newInner := pathRe.ReplaceAllStringFunc(inner, func(pmatch string) string {
			ps := pathRe.FindStringSubmatch(pmatch)
			if len(ps) < 4 {
				return pmatch
			}
			beforeD := ps[1]
			oldD := ps[2]
			afterD := ps[3]

			newD := ProcessGlyphPath(oldD, tx, ty)
			return fmt.Sprintf(`<path%s d="%s"%s>`, beforeD, newD, afterD)
		})

		// Rebuild <g> without the transform attribute
		newG := fmt.Sprintf(`<g id="%s"%s%s>%s</g>`, id, beforeTx, afterTx, newInner)
		return newG
	})

	err = os.WriteFile(outputFile, []byte(newContent), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outputFile, err)
		os.Exit(1)
	}
	fmt.Printf("Success! Wrote %s\n", outputFile)
	fmt.Println("All transforms baked in, paths converted to relative, each glyph ends with 'm 8 0'.")
}