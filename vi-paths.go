package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const program = "vi-paths"

func init() {
	log.SetFlags(0)
}

func main() {
	dryRun := flag.Bool("dry-run", false, "don't execute any operations, just print")
	flag.Parse()

	paths := flag.Args()
	if len(paths) == 0 {
		log.Fatalf("please provide a list of paths\nfor example using your shell's path globbing like ./**")
	}

	editor, ok := os.LookupEnv("EDITOR")
	if !ok {
		log.Fatalf("$EDITOR not set")
	}
	if _, err := exec.LookPath(editor); err != nil {
		log.Fatalf("$EDITOR %q not found in $PATH", editor)
	}

	if err := run(paths, editor, *dryRun); err != nil {
		log.Fatalf("running: %v", err)
	}
}

func run(before []string, editor string, dryRun bool) error {
	tmp, err := os.CreateTemp("", filepath.Base(program))
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	after, err := editPaths(tmp, editor, before)
	if err != nil {
		return fmt.Errorf("editing paths: %w", err)
	}
	if len(after) != len(before) {
		return fmt.Errorf("line count mismatch: before %d, after %d", len(before), len(after))
	}

	instructions, err := parseInstructions(before, after)
	if err != nil {
		return fmt.Errorf("parse instructions: %w", err)
	}
	for _, instruction := range instructions {
		log.Printf("%s", instruction)
		if dryRun {
			continue
		}
		if err := instruction.Execute(); err != nil {
			return fmt.Errorf("executing: %w", err)
		}
	}

	return nil
}

func editPaths(tmp *os.File, editor string, before []string) ([]string, error) {
	for _, name := range before {
		tmp.WriteString(name + "\n")
	}

	cmd := exec.Command(editor, tmp.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("running %q: %v", editor, err)
	}
	tmp.Seek(0, io.SeekStart)

	var after []string
	for r := bufio.NewScanner(tmp); r.Scan(); {
		after = append(after, r.Text())
	}

	return after, nil
}

func parseInstructions(before, after []string) ([]instruction, error) {
	// make sure we do the deepest operations first
	depth := func(path string) int { return strings.Count(path, string(filepath.Separator)) }
	multiSortStable(before, [][]string{after}, func(a, b string) bool {
		return depth(a) > depth(b)
	})

	const cmdCopy = "copy"

	var instructions []instruction
	for i := range before {
		switch before, after := strings.TrimSpace(before[i]), strings.TrimSpace(after[i]); {
		case strings.HasPrefix(after, fmt.Sprintf("%s ", cmdCopy)):
			instructions = append(instructions, copy{from: before, to: strings.TrimSpace(strings.TrimPrefix(after, cmdCopy))})
		case after == "":
			instructions = append(instructions, remove{name: before})
		case after != before:
			instructions = append(instructions, rename{before: before, after: after})
		}
	}

	return instructions, nil
}

type instruction interface {
	String() string
	Execute() error
}

type rename struct{ before, after string }

func (n rename) String() string { return fmt.Sprintf("rename %s\n    -> %s", n.before, n.after) }
func (n rename) Execute() error {
	if err := os.MkdirAll(filepath.Dir(n.after), 0755); err != nil {
		return fmt.Errorf("exe mkdirall: %w", err)
	}
	if err := os.Rename(n.before, n.after); err != nil {
		return fmt.Errorf("exe rename: %w", err)
	}
	return nil
}

type remove struct{ name string }

func (v remove) String() string { return fmt.Sprintf("remove %s", v.name) }
func (v remove) Execute() error {
	if err := os.RemoveAll(v.name); err != nil {
		return fmt.Errorf("exe remove all: %w", err)
	}
	return nil
}

type copy struct{ from, to string }

func (c copy) String() string { return fmt.Sprintf("copy %s\n  -> %s", c.from, c.to) }
func (c copy) Execute() error {
	stat, err := os.Stat(c.from)
	if err != nil {
		return fmt.Errorf("exe stat: %w", err)
	}
	if stat.IsDir() {
		if err := os.MkdirAll(c.to, stat.Mode()); err != nil {
			return fmt.Errorf("exe mkdirall: %w", err)
		}
		return nil
	}
	parentStat, err := os.Stat(filepath.Dir(c.from))
	if err != nil {
		return fmt.Errorf("exe stat: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(c.to), parentStat.Mode()); err != nil {
		return fmt.Errorf("exe mkdirall: %w", err)
	}
	input, err := os.ReadFile(c.from)
	if err != nil {
		return fmt.Errorf("exe read: %w", err)
	}
	if err := os.WriteFile(c.to, input, stat.Mode()); err != nil {
		return fmt.Errorf("exe write: %w", err)
	}
	return nil
}

type multiSortable[T any] struct {
	data  []T
	extra [][]T
	less  func(a, b T) bool
}

func (m *multiSortable[T]) Len() int           { return len(m.data) }
func (m *multiSortable[T]) Less(i, j int) bool { return m.less(m.data[i], m.data[j]) }
func (m *multiSortable[T]) Swap(i, j int) {
	for _, d := range append([][]T{m.data}, m.extra...) {
		d[i], d[j] = d[j], d[i]
	}
}

func multiSortStable[T any](data []T, extra [][]T, less func(a, b T) bool) {
	sort.Stable(&multiSortable[T]{data, extra, less})
}
