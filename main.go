package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"os"
	"path/filepath"
)

const INSTALLPATH = "~/bin/ollama"

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

func getLatestOllamaVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/ollama/ollama/releases/latest")
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return release.TagName, nil
}

func getDownloadURL(version string) string {
	return fmt.Sprintf("https://github.com/ollama/ollama/releases/download/%s/ollama-linux-amd64.tgz",
		version)
}

func installOllama(url string) error {
	tempFile := "/tmp/ollama.tgz"

	// Download the file to temp path
	curlCommand  := exec.Command("curl", "-L", url, "-o", tempFile)
	if err := curlCommand.Run(); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	// Create the bin directory if it doesn't exist
	homeDir, _ := os.UserHomeDir()
	binDir := filepath.Join(homeDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Extract the binary from the tgz file
	extractCommand := exec.Command("tar", "-xzf", tempFile, "-C", "/tmp")
	if err := extractCommand.Run(); err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	// Move the extracted binary to the final location
	finalPath := filepath.Join(binDir, "ollama")
	moveCommand := exec.Command("mv", "/tmp/ollama", finalPath)
	if err := moveCommand.Run(); err != nil {
		return fmt.Errorf("failed to move binary: %w", err)
	}

	// Make it executable
	chmodCommand := exec.Command("chmod", "+x", finalPath)
	if err := chmodCommand.Run(); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}
	
	// Clean up temporary file
	os.Remove(tempFile)
	
	fmt.Printf("Ollama installed successfully to %s\n", finalPath)
	return nil
}

func main() {
	version, err := getLatestOllamaVersion()
	if err != nil {
		fmt.Printf("Error getting latest version: %v\n", err)
		return
	}

	url := getDownloadURL(version)
	fmt.Printf("Latest Ollama version: %s\n", version)
	
	if err := installOllama(url); err != nil {
		fmt.Printf("Installation failed: %v\n", err)
		return
	}
}
