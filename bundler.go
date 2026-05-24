package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var bundleMagic = [8]byte{'L', 'L', 'B', 'N', 'D', 'L', '\x00', '\x00'}

type BundleData struct {
	Main string            `json:"m"`
	VFS  map[string]string `json:"v"`
}

type bundler struct {
	lexer  *Lexer
	parser *Parser
	files  map[string]bool
	vfs    map[string]string
}

func runBundle(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: ll -b <file.ll> [-o <output>]")
	}

	inputFile := args[0]
	outputFile := inputFile + ".bin"

	if len(args) >= 3 && args[1] == "-o" {
		outputFile = args[2]
	}

	absInput, err := filepath.Abs(inputFile)
	if err != nil {
		return fmt.Errorf("cannot resolve %s: %w", inputFile, err)
	}

	b := &bundler{
		lexer:  &Lexer{},
		parser: &Parser{},
		files:  make(map[string]bool),
		vfs:    make(map[string]string),
	}

	if err := b.collectDeps(absInput); err != nil {
		return err
	}

	mainData, err := os.ReadFile(absInput)
	if err != nil {
		return fmt.Errorf("cannot read main file: %w", err)
	}

	bd := BundleData{Main: string(mainData), VFS: b.vfs}
	bundleJSON, err := json.Marshal(bd)
	if err != nil {
		return fmt.Errorf("json error: %w", err)
	}

	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot get executable path: %w", err)
	}
	binaryData, err := os.ReadFile(selfPath)
	if err != nil {
		return fmt.Errorf("cannot read executable: %w", err)
	}

	out, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", outputFile, err)
	}
	defer out.Close()

	if err := out.Chmod(0755); err != nil {
		return fmt.Errorf("cannot set executable mode: %w", err)
	}

	if _, err := out.Write(binaryData); err != nil {
		return err
	}
	if _, err := out.Write(bundleJSON); err != nil {
		return err
	}

	lenBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBuf, uint64(len(bundleJSON)))
	if _, err := out.Write(lenBuf); err != nil {
		return err
	}
	magic := [8]byte{'L', 'L', 'B', 'N', 'D', 'L', '\x00', '\x00'}
	if _, err := out.Write(magic[:]); err != nil {
		return err
	}

	fmt.Printf("Bundled: %s -> %s\n", inputFile, outputFile)
	return nil
}

func (b *bundler) collectDeps(absPath string) error {
	if b.files[absPath] {
		return nil
	}
	b.files[absPath] = true

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", absPath, err)
	}

	tokens, err := b.lexer.Tokenize(string(data))
	if err != nil {
		return nil
	}
	ast, err := b.parser.Parse(tokens)
	if err != nil {
		return nil
	}

	baseDir := filepath.Dir(absPath)
	for _, expr := range ast {
		if err := b.walkForRequires(expr, baseDir); err != nil {
			return err
		}
	}

	return nil
}

func (b *bundler) walkForRequires(v Value, baseDir string) error {
	cons, ok := v.(*Cons)
	if !ok {
		return nil
	}

	if sym, ok := cons.Car.(*Sym); ok && (sym.Name == "require" || sym.Name == "include") {
		if args, ok := cons.Cdr.(*Cons); ok {
			if filename, ok := args.Car.(String); ok {
				depPath := resolveRequirePath(string(filename), baseDir)
				if depPath == "" {
					return fmt.Errorf("cannot find required file: %s (from %s)", string(filename), baseDir)
				}

				if err := b.collectDeps(depPath); err != nil {
					return err
				}

				data, err := os.ReadFile(depPath)
				if err != nil {
					return fmt.Errorf("cannot read dependency %s: %w", depPath, err)
				}
				b.vfs[string(filename)] = string(data)
			}
		}
	}

	if err := b.walkForRequires(cons.Car, baseDir); err != nil {
		return err
	}
	if err := b.walkForRequires(cons.Cdr, baseDir); err != nil {
		return err
	}
	return nil
}

func resolveRequirePath(path, baseDir string) string {
	if filepath.IsAbs(path) {
		if _, err := os.Stat(path); err == nil {
			return path
		}
		return ""
	}

	candidate := filepath.Join(baseDir, path)
	if _, err := os.Stat(candidate); err == nil {
		abs, _ := filepath.Abs(candidate)
		return abs
	}

	if _, err := os.Stat(path); err == nil {
		abs, _ := filepath.Abs(path)
		return abs
	}

	return ""
}

func readBundle() (*BundleData, error) {
	selfPath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(selfPath)
	if err != nil {
		return nil, err
	}
	if len(data) < 8+8 {
		return nil, fmt.Errorf("data too short")
	}
	n := len(data)
	magic := data[n-8:]
	if string(magic) != string(bundleMagic[:]) {
		return nil, fmt.Errorf("no bundle magic")
	}
	bundleLen := binary.LittleEndian.Uint64(data[n-16 : n-8])
	if uint64(n) < 16+bundleLen {
		return nil, fmt.Errorf("bundle data truncated")
	}
	start := uint64(n) - 16 - bundleLen
	raw := data[start : start+bundleLen]
	var bd BundleData
	if err := json.Unmarshal(raw, &bd); err != nil {
		return nil, fmt.Errorf("bundle JSON error: %w", err)
	}
	return &bd, nil
}
