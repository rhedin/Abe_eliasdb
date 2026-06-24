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
func ParsePath(d string) []Command {
	d = strings.TrimSpace(d)
	if d == "" {
		return nil
	}

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

// bake adds tx/ty to the appropriate coordinates.
func bake(cmds []Command, tx, ty float64) []Command {
	out := make([]Command, len(cmds))
	for i, c := range cmds {
		out[i].Cmd = c.Cmd
		out[i].Args = make([]float64, len(c.Args))
		copy(out[i].Args, c.Args)

		switch c.Cmd {
		case 'M', 'L', 'Q', 'T', 'C', 'S':
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
			if len(out[i].Args) >= 7 {
				out[i].Args[5] += tx
				out[i].Args[6] += ty
			}
		}
	}
	return out
}

// toRelative converts baked absolute commands to relative form.
// At the end it appends a relative move so the current point ends at (8, 0)
// relative to the glyph's origin.
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
				newCmd.Cmd = 'm'
				curX, curY = newArgs[0], newArgs[1]
				subStartX, subStartY = curX, curY
				firstMove = false
			} else {
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
			newArgs[0] -= curX
			newArgs[1] -= curY
			newArgs[2] -= curX
			newArgs[3] -= curY
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

	// Append a relative move so we end at (8, 0) relative to the glyph origin
	dx := 8 - curX
	dy := 0 - curY
	out = append(out, Command{Cmd: 'm', Args: []float64{dx, dy}})

	return out
}

func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}

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

func ProcessGlyphPath(originalD string, tx, ty float64) string {
	cmds := ParsePath(originalD)
	baked := bake(cmds, tx, ty)
	rel := toRelative(baked)
	return SerializePath(rel)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println("  SVG only:     go run normalize_glyphs.go input.html output-relative.html")
		fmt.Println("  Go map only:  go run normalize_glyphs.go input.html --go-map glyphpaths.go")
		fmt.Println("  Both:         go run normalize_glyphs.go input.html output-relative.html --go-map glyphpaths.go")
		return
	}

	inputFile := os.Args[1]
	var svgOut, goMapOut string

	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "--go-map" && i+1 < len(os.Args) {
			goMapOut = os.Args[i+1]
			i++
		} else if svgOut == "" {
			svgOut = os.Args[i]
		}
	}

	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", inputFile, err)
		os.Exit(1)
	}
	content := string(data)

	re := regexp.MustCompile(`(?s)<g\s+id="(ascii[0-9a-f]+)"([^>]*)transform="translate\(([0-9.-]+),([0-9.-]+)\)"([^>]*)>(.*?)</g>`)

	glyphMap := make(map[rune]string)

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

		var r rune
		if hexVal, err := strconv.ParseUint(id[5:], 16, 32); err == nil {
			r = rune(hexVal)
		}

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

			if r != 0 {
				glyphMap[r] = newD
			}
			return fmt.Sprintf(`<path%s d="%s"%s>`, beforeD, newD, afterD)
		})

		newG := fmt.Sprintf(`<g id="%s"%s%s>%s</g>`, id, beforeTx, afterTx, newInner)
		return newG
	})

	if svgOut != "" {
		err = os.WriteFile(svgOut, []byte(newContent), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing SVG %s: %v\n", svgOut, err)
			os.Exit(1)
		}
		fmt.Printf("Wrote SVG: %s\n", svgOut)
	}

	if goMapOut != "" {
		var b strings.Builder
		b.WriteString("package main\n\n")
		b.WriteString("// glyphPaths: baked + relative paths.\n")
		b.WriteString("// Each segment ends with a relative move so the current point\n")
		b.WriteString("// is left at the top-left of the next cell (relative to the glyph origin).\n")
		b.WriteString("var glyphPaths = map[rune]string{\n")

		for r := rune(32); r < 127; r++ {
			if s, ok := glyphMap[r]; ok {
				b.WriteString(fmt.Sprintf("\t%r: %q,\n", r, s))
			}
		}
		b.WriteString("}\n")

		err = os.WriteFile(goMapOut, []byte(b.String()), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing Go map %s: %v\n", goMapOut, err)
			os.Exit(1)
		}
		fmt.Printf("Wrote Go map: %s\n", goMapOut)
	}

	fmt.Println("Done.")
}
