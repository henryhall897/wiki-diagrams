//go:build mage

package main

import (
	"fmt"

	"github.com/magefile/mage/mg"
)

// Deps namespace coordinates dependency installation and verification for the entire repo.
type Deps mg.Namespace

// All runs all dependency checks sequentially.
func (Deps) All() error {
	fmt.Println("🔍 Ensuring all dependencies for Wiki-Diagrams are installed and verified...")

	steps := []struct {
		name string
		fn   func() error
	}{
		{"Go toolchain", func() error { return (Go{}).Deps() }},
		{"Mermaid CLI", func() error { return (Mermaid{}).Deps() }},
		{"Git configuration", func() error { return (Git{}).Deps() }},
	}

	for _, step := range steps {
		fmt.Printf("▶️  Starting: %s...\n", step.name)
		if err := step.fn(); err != nil {
			return fmt.Errorf("%s failed: %w", step.name, err)
		}
		fmt.Printf("✅ Completed: %s verified successfully.\n\n", step.name)
	}
	// After all deps are installed, run full verification
	fmt.Println("\n🔍 Running post-install verification pipeline...")
	mg.Deps(Deps.Verify)

	fmt.Println("🎨 All dependencies are installed, configured, and verified successfully.")
	return nil
}

// Verify runs lightweight verification (no installation) for all dependencies.
func (Deps) Verify() error {
	fmt.Println("🧭 Verifying installed dependencies for Wiki-Diagrams...")

	steps := []struct {
		name string
		fn   func() error
	}{
		{"Go toolchain", func() error { return (Go{}).Verify() }},
		{"Mermaid CLI", func() error { return (Mermaid{}).Verify() }},
		{"Git availability", func() error { return (Git{}).Verify() }},
	}

	for _, step := range steps {
		fmt.Printf("Checking: %s...\n", step.name)
		if err := step.fn(); err != nil {
			return fmt.Errorf("%s verification failed: %w", step.name, err)
		}
	}

	fmt.Println("\n✅ All dependency verifications passed successfully.")
	return nil
}

// Minimal runs only the essentials for CI environments.
// Skips installations requiring elevated privileges.
func (Deps) Minimal() error {
	fmt.Println("⚙️  Running minimal dependency check (CI mode)...")

	if err := (Mermaid{}).Verify(); err != nil {
		return fmt.Errorf("Mermaid CLI verification failed: %w", err)
	}
	if err := (Git{}).Verify(); err != nil {
		return fmt.Errorf("Git verification failed: %w", err)
	}

	fmt.Println("✅ Minimal dependency check passed.")
	return nil
}
