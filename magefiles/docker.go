//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Docker namespace groups all Docker-related verification and dependency tasks.
// In wiki-diagrams, Docker is used primarily for reproducible environments and secret mounting.
type Docker mg.Namespace

// Verify performs read-only checks for Docker, Buildx, and GitHub App key availability.
// It should never mutate system state.
func (Docker) Verify() error {
	fmt.Println("🔍 Verifying Docker environment for Wiki-Diagrams...")

	steps := []struct {
		name string
		fn   func() error
	}{
		{"Docker Engine & Buildx", verifyDockerEngine},
		{"GitHub App Private Key", verifyGitHubAppKey},
	}

	for _, step := range steps {
		fmt.Printf("→ %s...\n", step.name)
		if err := step.fn(); err != nil {
			return fmt.Errorf("%s check failed: %w", step.name, err)
		}
		fmt.Printf("   ✅ %s verified successfully.\n", step.name)
	}

	fmt.Println("✅ All Docker verifications passed successfully for Wiki-Diagrams.")
	return nil
}

// Deps is a self-healing task that ensures Docker and required secrets exist.
// After setup, it runs Verify() to confirm readiness.
func (Docker) Deps() error {
	fmt.Println("🧩 Ensuring Docker environment dependencies for Wiki-Diagrams...")

	if err := ensureDockerInstalled(); err != nil {
		return fmt.Errorf("failed ensuring Docker Engine: %w", err)
	}

	if err := ensureBuildxConfigured(); err != nil {
		return fmt.Errorf("failed ensuring Docker Buildx: %w", err)
	}

	if err := (Docker{}).Secrets(); err != nil {
		return fmt.Errorf("failed ensuring GitHub App key secret: %w", err)
	}

	fmt.Println("🧭 Running post-setup verification...")
	if err := (Docker{}).Verify(); err != nil {
		return fmt.Errorf("post-setup verification failed: %w", err)
	}

	fmt.Println("✅ Docker environment configured and verified successfully.")
	return nil
}

// --- Verification helpers ---

// verifyDockerEngine checks that Docker and Buildx are installed and reachable.
func verifyDockerEngine() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker binary not found in PATH — please install Docker Engine")
	}

	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker daemon not reachable: %w", err)
	}
	fmt.Printf("🐋 Docker Engine detected (version: %s)\n", strings.TrimSpace(string(out)))

	if err := exec.Command("docker", "buildx", "version").Run(); err != nil {
		return fmt.Errorf("docker buildx plugin missing — run: docker buildx install")
	}

	return nil
}

// --- Self-healing setup helpers ---

// ensureDockerInstalled installs Docker Engine and dependencies if not already installed.
func ensureDockerInstalled() error {
	if _, err := exec.LookPath("docker"); err == nil {
		return nil
	}

	fmt.Println("🐋 Installing Docker Engine (official repository)...")
	cmds := [][]string{
		{"sudo", "apt-get", "update", "-y"},
		{"sudo", "apt-get", "install", "-y", "ca-certificates", "curl", "gnupg"},
		{"sudo", "install", "-m", "0755", "-d", "/etc/apt/keyrings"},
		{"bash", "-c", "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg"},
		{"bash", "-c", "echo \"deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable\" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null"},
		{"sudo", "apt-get", "update", "-y"},
		{"sudo", "apt-get", "install", "-y", "docker-ce", "docker-ce-cli", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed running %v: %w", args, err)
		}
	}

	fmt.Println("✅ Docker Engine installed successfully.")
	return nil
}

// ensureBuildxConfigured ensures Docker Buildx is installed and functional.
func ensureBuildxConfigured() error {
	if err := exec.Command("docker", "buildx", "version").Run(); err == nil {
		return nil
	}
	fmt.Println("⚙️  Installing Docker Buildx plugin...")
	return sh.RunV("docker", "buildx", "install")
}

// --- Secrets management ---

// Secrets ensures the GitHub App private key is available for local or Swarm use.
func (Docker) Secrets() error {
	fmt.Println("🔧 Ensuring GitHub App private key is available...")

	const secretName = "wiki_diagram_app_key"
	const swarmSecretPath = "/run/secrets/" + secretName

	// Case 1: If Swarm secret already exists
	if _, err := os.Stat(swarmSecretPath); err == nil {
		fmt.Println("✅ GitHub App key available via Docker secret:", swarmSecretPath)
		return nil
	}

	// Case 2: Look for local .pem file
	home, _ := os.UserHomeDir()
	searchDir := filepath.Join(home, ".config", "github-apps")
	matches, err := filepath.Glob(filepath.Join(searchDir, "wiki-diagram-publisher*.pem"))
	if err != nil {
		return fmt.Errorf("error searching for GitHub App key in %s: %w", searchDir, err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("no local GitHub App key found in %s — please download from GitHub App settings", searchDir)
	}

	localKey := matches[len(matches)-1]
	fmt.Printf("📦 Found local GitHub App key: %s\n", filepath.Base(localKey))

	// Case 3: Only create a Docker secret if Swarm mode is active
	out, _ := exec.Command("docker", "info", "--format", "{{.Swarm.ControlAvailable}}").Output()
	if strings.TrimSpace(string(out)) == "true" {
		fmt.Println("🐝 Swarm mode detected — creating Docker secret...")
		if err := sh.RunV("docker", "secret", "create", secretName, localKey); err != nil {
			if strings.Contains(err.Error(), "already exists") {
				fmt.Println("ℹ️  Docker secret already exists, skipping creation.")
				return nil
			}
			return fmt.Errorf("failed to create Docker secret: %w", err)
		}
		fmt.Println("✅ Docker secret created successfully.")
	} else {
		fmt.Println("💻 Running in non-Swarm mode — using local key directly for verification.")
	}

	return nil
}

// verifyGitHubAppKey searches for the GitHub App private key in known locations.
// It supports date-suffixed filenames like wiki-diagram-publisher.YYYY-MM-DD.private-key.pem.
func verifyGitHubAppKey() error {
	const secretPath = "/run/secrets/wiki_diagram_app_key"

	// 1️⃣ Environment variable override
	if keyPath := os.Getenv("WIKI_APP_PRIVATE_KEY_PATH"); keyPath != "" {
		if _, err := os.Stat(keyPath); err != nil {
			return fmt.Errorf("GitHub App key missing at %s: %w", keyPath, err)
		}
		fmt.Println("📦 Using GitHub App key from environment variable:", keyPath)
		return nil
	}

	// 2️⃣ Docker secret mount
	if _, err := os.Stat(secretPath); err == nil {
		fmt.Println("📦 Using GitHub App key from Docker secret:", secretPath)
		return nil
	}

	// 3️⃣ Local key search
	home, _ := os.UserHomeDir()
	searchDir := filepath.Join(home, ".config", "github-apps")
	fmt.Println("🔍 Searching for key files in:", searchDir)

	matches, err := filepath.Glob(filepath.Join(searchDir, "wiki-diagram-publisher*.pem"))
	if err != nil {
		return fmt.Errorf("error searching for GitHub App key in %s: %w", searchDir, err)
	}
	if len(matches) == 0 {
		return fmt.Errorf(`no GitHub App key found — expected one of:
  - env var: WIKI_APP_PRIVATE_KEY_PATH
  - Docker secret: %s
  - local file: ~/.config/github-apps/wiki-diagram-publisher*.pem`, secretPath)
	}

	latestKey := matches[len(matches)-1]
	if _, err := os.Stat(latestKey); err != nil {
		return fmt.Errorf("GitHub App key found but unreadable: %w", err)
	}

	fmt.Println("💻 Using local GitHub App key:", latestKey)
	return nil
}
