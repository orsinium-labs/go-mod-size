package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

type Dep struct {
	name       string
	deps       []string
	directSize int64
	totalSize  int64
	resolved   bool
	approx     bool
}

func run() error {
	cmd := exec.Command("go", "mod", "graph")
	if len(os.Args) == 2 {
		cmd.Dir = os.Args[1]
	}
	rawStdout, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("run go mod graph: %v", err)
	}
	stdout := string(rawStdout)

	// Collect direct sizes
	deps := map[string]*Dep{}
	for line := range strings.SplitSeq(stdout, "\n") {
		parent, child, found := strings.Cut(line, " ")
		if !found {
			continue
		}
		dep := deps[parent]
		if dep != nil {
			dep.deps = append(dep.deps, child)
		} else {
			size, _ := getSize(parent)
			deps[parent] = &Dep{
				name:       parent,
				deps:       []string{child},
				directSize: size,
			}
		}
	}

	// Accumulate total sizes
	allResolved := false
	cycleDetected := false
	attempts := 0
	for !allResolved {
		attempts += 1
		if attempts > 10_000 {
			return errors.New("max dependency depth is reached")
		}
		allResolved = true
		someResolved := false
		for _, parent := range deps {
			if parent.resolved {
				continue
			}

			// If any subdependency doesn't have its total size resolved,
			// don't resolve this one just yet.
			depResolved := true
			for _, childName := range parent.deps {
				child := deps[childName]
				if child != nil && !child.resolved {
					depResolved = false
					break
				}
			}
			if !depResolved {
				allResolved = false
				if cycleDetected {
					continue
				}
			}

			someResolved = true
			parent.approx = !depResolved
			parent.resolved = true
			parent.totalSize = parent.directSize
			for _, childName := range parent.deps {
				child := deps[childName]
				if child != nil {
					parent.totalSize += child.totalSize
				}
			}
			if !depResolved {
				break
			}
		}
		if !someResolved {
			cycleDetected = true
		}
	}

	// Sort.
	sorted := make([]Dep, 0, len(deps))
	for _, dep := range deps {
		sorted = append(sorted, *dep)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].totalSize > sorted[j].totalSize
	})

	// Print.
	for _, dep := range sorted {
		direct := formatSize(dep.directSize)
		total := formatSize(dep.totalSize)
		if dep.approx {
			total = "~" + total
		}
		fmt.Printf("%-80s %10s %10s\n", dep.name, direct, total)
	}

	return nil
}

func getSize(name string) (int64, error) {
	gopath, found := os.LookupEnv("GOPATH")
	if !found {
		home, err := os.UserHomeDir()
		if err != nil {
			return 0, err
		}
		gopath = path.Join(home, "go")
	}
	depPath := path.Join(gopath, "pkg", "mod", name)
	var size int64
	err := filepath.Walk(depPath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func formatSize(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(1024), 0
	for n := b / 1024; n >= 1024; n /= 1024 {
		div *= 1024
		exp++
	}
	unit := "KMGTPE"[exp]
	return fmt.Sprintf("%.1f%c", float64(b)/float64(div), unit)
}
