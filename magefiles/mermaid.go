//go:build mage

package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Mermaid namespace groups all Mermaid CLI–related tasks.
type Mermaid mg.Namespace

// TargetMermaidVersion defines the pinned version for reproducible builds.
const TargetMermaidVersion = "10.9.0"

// All runs the full end-to-end pipeline:
// 1. Ensures all dependencies (system, Go, Git, Mermaid CLI, etc.)
// 2. Cleans previously generated diagrams
// 3. Rebuilds all diagrams (Markdown → MMD → SVG)
func (Mermaid) All() error {
	fmt.Println("🌊 Running full Mermaid pipeline: deps → clean → renderall")

	// Step 1: Ensure all dependencies
	mg.Deps(Deps.All)

	// Step 2: Clean old diagrams
	mg.Deps(Diagrams.Clean)

	// Step 3: Rebuild all diagrams
	mg.Deps(Diagrams.RenderAll)

	fmt.Println("✅ Mermaid:all pipeline completed successfully.")
	return nil
}

// Verify checks that the Mermaid CLI is installed and matches the target version.
func (Mermaid) Verify() error {
	fmt.Println("Verifying Mermaid CLI installation...")

	out, err := exec.Command("mmdc", "--version").CombinedOutput()
	if err != nil {
		return errors.New("❌ Mermaid CLI not found in PATH. Install it with:\n   npm install -g @mermaid-js/mermaid-cli@" + TargetMermaidVersion)
	}

	version := strings.TrimSpace(string(out))
	if !strings.Contains(version, TargetMermaidVersion) {
		return fmt.Errorf("❌ Mermaid CLI version mismatch: found '%s', expected %s", version, TargetMermaidVersion)
	}

	fmt.Printf("✅ Mermaid CLI %s verified successfully.\n", TargetMermaidVersion)
	return nil
}

// Deps ensures that all Mermaid-related dependencies are installed and verified.
func (Mermaid) Deps() error {
	fmt.Println("Ensuring Mermaid CLI dependencies...")

	// Step 1: Verify system libraries for headless Chromium rendering
	if err := (Mermaid{}).VerifySystemLibs(); err != nil {
		return fmt.Errorf("system library verification failed: %w", err)
	}

	// Step 2: Verify Mermaid CLI installation
	if err := (Mermaid{}).Verify(); err == nil {
		fmt.Println("✅ Mermaid CLI already installed and up to date.")
		return nil
	}

	// Step 3: Install Mermaid CLI globally
	fmt.Printf("Installing Mermaid CLI %s globally via npm...\n", TargetMermaidVersion)
	if err := sh.RunV("npm", "install", "-g",
		fmt.Sprintf("@mermaid-js/mermaid-cli@%s", TargetMermaidVersion)); err != nil {
		return fmt.Errorf("failed to install Mermaid CLI %s: %w", TargetMermaidVersion, err)
	}

	// Step 4: Re-verify installation
	fmt.Println("Re-verifying Mermaid installation...")
	if err := (Mermaid{}).Verify(); err != nil {
		return fmt.Errorf("Mermaid CLI installation did not verify successfully: %w", err)
	}

	fmt.Printf("✅ Mermaid CLI %s successfully installed and verified.\n", TargetMermaidVersion)
	return nil
}

// Version prints the currently installed Mermaid CLI version.
func (Mermaid) Version() error {
	out, err := exec.Command("mmdc", "--version").CombinedOutput()
	if err != nil {
		return errors.New("Mermaid CLI not found in PATH.")
	}
	fmt.Printf("Mermaid CLI version: %s\n", strings.TrimSpace(string(out)))
	return nil
}

// VerifySystemLibs ensures system libraries required for headless Chromium are installed.
func (Mermaid) VerifySystemLibs() error {
	fmt.Println("🔍 Verifying required system libraries for Mermaid CLI...")

	libs := []string{
		"libatk1.0-0t64", "libatk-bridge2.0-0t64", "libcups2t64", "libdrm2", "libxkbcommon0",
		"libxdamage1", "libxfixes3", "libxrandr2", "libasound2t64", "libatspi2.0-0t64",
		"libpangocairo-1.0-0", "libpango-1.0-0", "libcairo2", "libgbm1", "libnss3",
		"libxshmfence1", "libxcomposite1", "libxext6", "libx11-6", "libx11-xcb1",
		"libxcb1", "libxrender1", "fonts-liberation", "libgtk-3-0t64",
	}

	missing := []string{}
	for _, pkg := range libs {
		cmd := exec.Command("dpkg", "-s", pkg)
		if err := cmd.Run(); err != nil {
			missing = append(missing, pkg)
		}
	}

	if len(missing) == 0 {
		fmt.Println("✅ All required system libraries are installed.")
		return nil
	}

	fmt.Printf("⚠️  Missing %d libraries:\n", len(missing))
	for _, pkg := range missing {
		fmt.Printf("   - %s\n", pkg)
	}

	fmt.Println("\nInstalling missing libraries (requires sudo)...")

	// Step 1: Update package lists
	if err := sh.RunV("sudo", "apt-get", "update", "-q"); err != nil {
		return fmt.Errorf("failed to update package lists: %w", err)
	}

	// Step 2: Install missing libraries
	args := append([]string{"apt-get", "install", "-y"}, missing...)
	if err := sh.RunV("sudo", args...); err != nil {
		return fmt.Errorf("failed to install required libraries: %w", err)
	}

	fmt.Println("✅ All required libraries installed successfully.")
	return nil

}
