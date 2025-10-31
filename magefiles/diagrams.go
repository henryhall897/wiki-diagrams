//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
)

// Directory structure
var (
	srcMDDir  = "assets/diagrams/src/"
	genMMDDir = "assets/diagrams/gen/mmd"
	genPNGDir = "assets/diagrams/gen/png"
	//mermaidCmd = "mmdc"
	outputExt = "png"
)

// Diagrams namespace handles all diagram generation tasks.
type Diagrams mg.Namespace

// RenderAll extracts .mmd from all .md files, then generates .svg from them.
func (Diagrams) RenderAll() error {
	fmt.Println("🎨 Rendering all diagrams from Markdown sources...")

	if err := ensureDir(genMMDDir); err != nil {
		return err
	}
	if err := ensureDir(genPNGDir); err != nil {
		return err
	}

	// Walk through Markdown source files
	return filepath.Walk(srcMDDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		base := strings.TrimSuffix(filepath.Base(path), ".md")
		mmdPath := filepath.Join(genMMDDir, base+".mmd")
		svgPath := filepath.Join(genPNGDir, base+"."+outputExt)

		fmt.Printf("→ %s\n", path)
		if err := extractMMD(path, mmdPath); err != nil {
			return fmt.Errorf("failed to extract MMD from %s: %w", path, err)
		}
		if err := renderFile(mmdPath, svgPath); err != nil {
			return fmt.Errorf("failed to render SVG for %s: %w", base, err)
		}
		fmt.Printf("✅ Generated: %s\n", svgPath)
		return nil
	})
}

// RenderOne regenerates a specific diagram by name (without extension).
func (Diagrams) RenderOne(name string) error {
	mdPath := filepath.Join(srcMDDir, name+".md")
	mmdPath := filepath.Join(genMMDDir, name+".mmd")
	svgPath := filepath.Join(genPNGDir, name+"."+outputExt)

	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		return fmt.Errorf("markdown file not found: %s", mdPath)
	}

	fmt.Printf("🎯 Rendering %s.md → %s → %s\n", name, mmdPath, svgPath)
	if err := extractMMD(mdPath, mmdPath); err != nil {
		return err
	}
	return renderFile(mmdPath, svgPath)
}

// Clean removes all generated diagram outputs.
func (Diagrams) Clean() error {
	fmt.Println("🧹 Cleaning generated diagrams...")
	if err := os.RemoveAll(genMMDDir); err != nil {
		return err
	}
	if err := os.RemoveAll(genPNGDir); err != nil {
		return err
	}
	return nil
}

// extractMMD parses Markdown, extracts ```mermaid blocks, and writes to a .mmd file.
func extractMMD(mdPath, mmdPath string) error {
	data, err := os.ReadFile(mdPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	var inBlock bool
	var output []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "```mermaid"):
			inBlock = true
			continue
		case strings.HasPrefix(trimmed, "```") && inBlock:
			inBlock = false
			continue
		}
		if inBlock {
			output = append(output, line)
		}
	}

	if len(output) == 0 {
		return fmt.Errorf("no mermaid block found in %s", mdPath)
	}

	if err := ensureDir(filepath.Dir(mmdPath)); err != nil {
		return err
	}
	return os.WriteFile(mmdPath, []byte(strings.Join(output, "\n")), 0644)
}

func renderFile(input, output string) error {
	puppeteerConfig := "assets/diagrams/puppeteer-config.json"
	mermaidConfig := "assets/diagrams/mermaid-config.json"

	cmd := exec.Command(
		"mmdc",
		"-i", input,
		"-o", output,
		"--configFile", mermaidConfig,
		"--puppeteerConfigFile", puppeteerConfig,
		"--backgroundColor", "#1B1B2F",
	)

	// Stream live logs to terminal
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	fmt.Printf("📘 Rendering with configs:\n   - %s\n   - %s\n", mermaidConfig, puppeteerConfig)

	return cmd.Run()
}

// ensureDir ensures a directory exists.
func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}
