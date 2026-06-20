package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// caseSensitiveName must match exactly what is in helpers.go
func caseSensitiveName(part, kind, suffix string) string {
	h := fnv.New32a()
	h.Write([]byte(kind))
	hashStr := fmt.Sprintf("%08x", h.Sum32())[:4]
	return part + kind + hashStr + suffix
}

var dryRun = flag.Bool("dry-run", true, "Only show what would be renamed (default: true)")

func main() {
	flag.Parse()

	dbDir := "db"
	if len(flag.Args()) > 0 {
		dbDir = flag.Args()[0]
	}

	fmt.Printf("=== EliasDB storage migration ===\n")
	fmt.Printf("Directory: %s\nDry-run: %v\n\n", dbDir, *dryRun)
	fmt.Println("WARNING: Make sure you have a backup of the 'db/' folder before running with -dry-run=false!")

	// Corrected base suffixes (longer ones first)
	baseSuffixes := []string{
		".nodeidx",
		".nodes",
		".edgeidx",
		".edges",
	}

	renamed := 0

	err := filepath.WalkDir(dbDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		name := d.Name()

		if !strings.HasPrefix(name, "main") {
			return nil
		}

		for _, suffix := range baseSuffixes {
			if idx := strings.Index(name, suffix); idx != -1 {
				oldBase := name[:idx+len(suffix)]
				remainder := name[idx+len(suffix):]

				kind := strings.TrimPrefix(oldBase, "main")
				kind = strings.TrimSuffix(kind, suffix)

				part := "main"

				newBase := caseSensitiveName(part, kind, suffix)
				newName := newBase + remainder

				if oldBase == newBase {
					break
				}

				oldPath := path
				newPath := filepath.Join(filepath.Dir(path), newName)

				fmt.Printf("  %s  →  %s\n", name, newName)

				if !*dryRun {
					if err := os.Rename(oldPath, newPath); err != nil {
						fmt.Printf("    ERROR: %v\n", err)
					} else {
						renamed++
					}
				} else {
					renamed++
				}

				break
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Walk error: %v\n", err)
		return
	}

	fmt.Printf("\nMigration %s complete — %d files affected.\n",
		map[bool]string{true: "DRY-RUN", false: "EXECUTED"}[*dryRun], renamed)

	if *dryRun {
		fmt.Println("\nTo actually perform the rename, run:")
		fmt.Println("    go run migrate.go -dry-run=false")
	}
}
