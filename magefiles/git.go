//go:build mage

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
)

// Git namespace handles Git-related verification and configuration tasks.
type Git mg.Namespace

// Verify ensures Git is installed and available in PATH.
func (Git) Verify() error {
	fmt.Println("Verifying Git installation...")
	out, err := exec.Command("git", "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("❌ Git not found in PATH: %w", err)
	}

	version := strings.TrimSpace(string(out))
	fmt.Println("✅", version)
	return nil
}

// Config ensures that user.name and user.email are set for local commits.
func (Git) Config() error {
	fmt.Println("Checking Git user configuration...")

	name, _ := exec.Command("git", "config", "--global", "user.name").Output()
	email, _ := exec.Command("git", "config", "--global", "user.email").Output()

	if strings.TrimSpace(string(name)) == "" || strings.TrimSpace(string(email)) == "" {
		fmt.Println("⚠️ Git user.name or user.email is not configured.")
		fmt.Println("To set globally, run:")
		fmt.Println("  git config --global user.name \"Your Name\"")
		fmt.Println("  git config --global user.email \"you@example.com\"")
		return nil
	}

	fmt.Printf("✅ Git user configured as %s <%s>\n",
		strings.TrimSpace(string(name)),
		strings.TrimSpace(string(email)),
	)
	return nil
}

// Deps ensures Git is available and properly configured.
func (Git) Deps() error {
	if err := (Git{}).Verify(); err != nil {
		return err
	}
	if err := (Git{}).Config(); err != nil {
		return err
	}
	return nil
}

// CheckRemote optionally verifies that GitHub is reachable and the remote URL is valid.
func (Git) CheckRemote() error {
	fmt.Println("Verifying GitHub connectivity and remote access...")
	cmd := exec.Command("git", "ls-remote", "--heads", "origin")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("❌ Failed to reach GitHub remote: %w\n%s", err, string(out))
	}
	fmt.Printf("✅ GitHub remote accessible (%d bytes returned)\n", len(out))
	return nil
}

// Info prints details about the current repository and branch.
func (Git) Info() error {
	fmt.Println("📂 Git repository information:")
	branch, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	remote, _ := exec.Command("git", "config", "--get", "remote.origin.url").Output()
	lastCommit, _ := exec.Command("git", "log", "-1", "--pretty=format:%h - %s (%cr)").Output()

	fmt.Printf("Branch: %s\n", strings.TrimSpace(string(branch)))
	fmt.Printf("Remote: %s\n", strings.TrimSpace(string(remote)))
	fmt.Printf("Last Commit: %s\n", strings.TrimSpace(string(lastCommit)))
	fmt.Printf("Checked: %s\n", time.Now().Format(time.RFC1123))
	return nil
}
