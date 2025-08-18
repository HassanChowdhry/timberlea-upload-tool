/*
Package main provides an installer for the Ollama CLI tool.
It downloads the latest version from GitHub releases and installs it
to the user's ~/bin directory, automatically updating shell configuration
to include the binary in PATH.
*/
package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	/* GitHub API configuration */
	githubAPIURL = "https://api.github.com/repos/ollama/ollama/releases/latest"

	/* File permissions */
	executableMode = 0755
	configFileMode = 0644

	/* HTTP timeout */
	httpTimeout = 30 * time.Second

	/* Temporary directory name */
	tempDirName = "ollama-extract"
)

/* Platform-specific configuration */
type PlatformConfig struct {
	downloadURLTemplate string
	tempFileName        string
	installPath         string
	binaryName          string
}

/*
GitHubRelease represents the structure of a GitHub release API response.
It contains the tag name which corresponds to the version number.
*/
type GitHubRelease struct {
	/* TagName is the git tag associated with the release (e.g., "v0.1.20") */
	TagName string `json:"tag_name"`
}

/*
getLatestOllamaVersion fetches the latest Ollama version from the GitHub API.
It makes an HTTP GET request to the GitHub releases API and parses the response
to extract the tag name of the latest release.

Parameters:
  - ctx: Context for request cancellation and timeout

Returns:
  - string: The version tag (e.g., "v0.1.20")
  - error: Any error that occurred during the API call or response parsing
*/
func getLatestOllamaVersion(ctx context.Context) (string, error) {
	client := &http.Client{
		Timeout: httpTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", githubAPIURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return release.TagName, nil
}

/*
getPlatformConfig returns the platform-specific configuration based on the current OS.
It determines the appropriate download URL template, file extensions, and paths
for Windows and Linux platforms.

Returns:
  - PlatformConfig: Configuration struct with platform-specific settings
*/
func getPlatformConfig() PlatformConfig {
	switch runtime.GOOS {
	case "windows":
		return PlatformConfig{
			downloadURLTemplate: "https://github.com/ollama/ollama/releases/download/%s/ollama-windows-amd64.zip",
			tempFileName:        "ollama.zip",
			installPath:         "~/AppData/Local/Programs/Ollama/ollama.exe",
			binaryName:          "ollama.exe",
		}
	case "linux":
		return PlatformConfig{
			downloadURLTemplate: "https://github.com/ollama/ollama/releases/download/%s/ollama-linux-amd64.tgz",
			tempFileName:        "ollama.tgz",
			installPath:         "~/bin/ollama",
			binaryName:          "ollama",
		}
	case "darwin":
		return PlatformConfig{
			downloadURLTemplate: "https://github.com/ollama/ollama/releases/download/%s/ollama-darwin.zip",
			tempFileName:        "ollama.zip",
			installPath:         "~/bin/ollama",
			binaryName:          "ollama",
		}
	default:
		// Default to Linux
		return PlatformConfig{
			downloadURLTemplate: "https://github.com/ollama/ollama/releases/download/%s/ollama-linux-amd64.tgz",
			tempFileName:        "ollama.tgz",
			installPath:         "~/bin/ollama",
			binaryName:          "ollama",
		}
	}
}

/*
getDownloadURL constructs the download URL for a specific Ollama version.
It formats the GitHub releases download URL template with the provided version
using platform-specific configuration.

Parameters:
  - version: The version tag (e.g., "v0.1.20")

Returns:
  - string: The complete download URL for the current platform
*/
func getDownloadURL(version string) string {
	config := getPlatformConfig()
	return fmt.Sprintf(config.downloadURLTemplate, version)
}

/*
installOllama downloads and installs Ollama to the user's bin directory.
It performs the complete installation process including:
  - Downloading the binary archive from the provided URL
  - Creating the ~/bin directory if it doesn't exist
  - Extracting and installing the binary
  - Making the binary executable
  - Updating shell configuration files to include ~/bin in PATH
  - Cleaning up temporary files

Parameters:
  - ctx: Context for request cancellation and timeout
  - url: The download URL for the Ollama binary archive

Returns:
  - error: Any error that occurred during the installation process
*/
func installOllama(ctx context.Context, url string) error {
	config := getPlatformConfig()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	tempFile := filepath.Join(homeDir, config.tempFileName)

	/* Ensure cleanup of temporary files */
	defer func() {
		os.Remove(tempFile)
	}()

	/* Download the file */
	if err := downloadFile(ctx, url, tempFile); err != nil {
		return fmt.Errorf("failed to download Ollama: %w", err)
	}

	/* For all platforms, use the extraction method */
	tempDir := filepath.Join(homeDir, tempDirName)
	
	/* Determine the installation directory based on platform */
	var binDir string
	var finalPath string
	if runtime.GOOS == "windows" {
		binDir = filepath.Join(homeDir, "AppData", "Local", "Programs", "Ollama")
		finalPath = filepath.Join(binDir, config.binaryName)
	} else {
		binDir = filepath.Join(homeDir, "bin")
		finalPath = filepath.Join(binDir, config.binaryName)
	}

	/* Ensure cleanup of temporary directory */
	defer func() {
		os.RemoveAll(tempDir)
	}()

	/* Create the bin directory if it doesn't exist */
	if err := os.MkdirAll(binDir, executableMode); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	/* Extract and install the binary */
	if err := extractAndInstall(tempFile, tempDir, finalPath, config); err != nil {
		return fmt.Errorf("failed to extract and install: %w", err)
	}

	/* Update PATH in shell configuration (skip for Windows as it uses standard location) */
	if runtime.GOOS != "windows" {
		if err := updatePath(homeDir); err != nil {
			fmt.Printf("Warning: Failed to update PATH: %v\n", err)
		}
	} else {
		fmt.Printf("Using standard Windows Ollama location (already in PATH)\n")
	}

	fmt.Printf("Ollama installed successfully to %s\n", finalPath)
	fmt.Printf("Please restart your terminal OR log out and log back in to use the new version\n")
	return nil
}

/*
downloadFile downloads a file from the given URL to the specified path.
It uses Go's native HTTP client with progress indicators and follows redirects.
The download progress is displayed to stdout.

Parameters:
  - ctx: Context for request cancellation and timeout
  - url: The URL to download from
  - filePath: The local file path where the download should be saved

Returns:
  - error: Any error that occurred during the download process
*/
func downloadFile(ctx context.Context, url, filePath string) error {
	fmt.Printf("Downloading Ollama from %s...\n", url)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: httpTimeout,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create the output file
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Get file size for progress tracking
	fileSize := resp.ContentLength

	// Create a progress reader
	progressReader := &ProgressReader{
		Reader: resp.Body,
		Total:  fileSize,
	}

	// Copy with progress
	_, err = io.Copy(out, progressReader)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	fmt.Println() // New line after progress
	return nil
}

/*
ProgressReader wraps an io.Reader to provide download progress feedback.
*/
type ProgressReader struct {
	Reader    io.Reader
	Total     int64
	BytesRead int64
}

/*
Read implements io.Reader interface and tracks progress.
*/
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.BytesRead += int64(n)

	if pr.Total > 0 {
		percentage := float64(pr.BytesRead) / float64(pr.Total) * 100
		fmt.Printf("\rProgress: %.1f%% (%d/%d bytes)", percentage, pr.BytesRead, pr.Total)
	} else {
		fmt.Printf("\rDownloaded: %d bytes", pr.BytesRead)
	}

	return n, err
}

/*
extractAndInstall extracts the downloaded archive and installs the binary.
It performs the following steps:
  - Creates a temporary extraction directory
  - Extracts the archive (ZIP for Windows, TGZ for Linux) using Go native libraries
  - Copies the extracted binary to the final installation path
  - Sets executable permissions on the binary

Parameters:
  - archivePath: Path to the downloaded archive
  - tempDir: Temporary directory for extraction
  - finalPath: Final installation path for the binary
  - config: Platform-specific configuration

Returns:
  - error: Any error that occurred during extraction or installation
*/
func extractAndInstall(archivePath, tempDir, finalPath string, config PlatformConfig) error {
	/* Clean up and create temporary extraction directory */
	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("failed to remove existing temp directory: %w", err)
	}

	if err := os.MkdirAll(tempDir, executableMode); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	fmt.Printf("Extracting Ollama binary...\n")

	/* Extract based on file type */
	var sourcePath string
	var err error

	if strings.HasSuffix(config.tempFileName, ".zip") {
		sourcePath, err = extractZip(archivePath, tempDir, config.binaryName)
	} else {
		sourcePath, err = extractTarGz(archivePath, tempDir)
	}

	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	/* Copy the extracted binary to the final location */
	if err := copyFile(sourcePath, finalPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	/* Make it executable */
	if err := os.Chmod(finalPath, executableMode); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	return nil
}

/*
extractZip extracts a ZIP archive and returns the path to the binary.
This is used for Windows and macOS downloads.

Parameters:
  - archivePath: Path to the ZIP file
  - tempDir: Directory to extract to
  - binaryName: Name of the binary to find

Returns:
  - string: Path to the extracted binary
  - error: Any error that occurred during extraction
*/
func extractZip(archivePath, tempDir, binaryName string) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	var binaryPath string

	for _, file := range reader.File {
		/* Create the file path */
		path := filepath.Join(tempDir, file.Name)

		/* Ensure we don't extract outside of tempDir */
		if !strings.HasPrefix(path, filepath.Clean(tempDir)+string(os.PathSeparator)) {
			continue
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.FileInfo().Mode())
			continue
		}

		/* Create parent directories */
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}

		/* Extract file */
		fileReader, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("failed to open file in zip: %w", err)
		}

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			fileReader.Close()
			return "", fmt.Errorf("failed to create target file: %w", err)
		}

		_, err = io.Copy(targetFile, fileReader)
		fileReader.Close()
		targetFile.Close()

		if err != nil {
			return "", fmt.Errorf("failed to copy file: %w", err)
		}

		/* Check if this is the binary we're looking for */
		if filepath.Base(path) == binaryName {
			binaryPath = path
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("binary %s not found in zip archive", binaryName)
	}

	return binaryPath, nil
}

/*
extractTarGz extracts a tar.gz archive and returns the path to the binary.
This is used for Linux downloads. Uses pure Go implementation.

Parameters:
  - archivePath: Path to the tar.gz file
  - tempDir: Directory to extract to

Returns:
  - string: Path to the extracted binary
  - error: Any error that occurred during extraction
*/
func extractTarGz(archivePath, tempDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var binaryPath string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Create the file path
		path := filepath.Join(tempDir, header.Name)

		// Ensure we don't extract outside of tempDir (security check)
		if !strings.HasPrefix(path, filepath.Clean(tempDir)+string(os.PathSeparator)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(path, os.FileMode(header.Mode)); err != nil {
				return "", fmt.Errorf("failed to create directory %s: %w", path, err)
			}

		case tar.TypeReg:
			// Create parent directories
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return "", fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create and write file
			outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return "", fmt.Errorf("failed to create file %s: %w", path, err)
			}

			_, err = io.Copy(outFile, tarReader)
			outFile.Close()

			if err != nil {
				return "", fmt.Errorf("failed to write file %s: %w", path, err)
			}

			// Check if this is the binary we're looking for
			if strings.HasSuffix(path, "/bin/ollama") || filepath.Base(path) == "ollama" {
				binaryPath = path
			}
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("ollama binary not found in tar.gz archive")
	}

	return binaryPath, nil
}

/*
copyFile copies a file from source to destination.

Parameters:
  - src: Source file path
  - dst: Destination file path

Returns:
  - error: Any error that occurred during copying
*/
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

/*
updatePath adds the ~/bin directory to PATH based on the operating system.
For Windows, it updates the user PATH environment variable using PowerShell.
For Unix systems, it updates shell configuration files.

Parameters:
  - homeDir: The user's home directory path

Returns:
  - error: Any error that occurred during the PATH update process
*/
func updatePath(homeDir string) error {
	if runtime.GOOS == "windows" {
		return updateWindowsPath(homeDir)
	}
	return updateUnixPath(homeDir)
}

/*
updateWindowsPath adds the ~/bin directory to the Windows user PATH environment variable.
It uses PowerShell to safely update the PATH without truncation issues.

Parameters:
  - homeDir: The user's home directory path

Returns:
  - error: Any error that occurred during the PATH update process
*/
func updateWindowsPath(homeDir string) error {
	binDir := filepath.Join(homeDir, "bin")

	// Check if already in PATH
	currentPath := os.Getenv("PATH")
	if strings.Contains(currentPath, binDir) {
		fmt.Printf("PATH already contains %s\n", binDir)
		return nil
	}

	// Use PowerShell to safely update the user PATH environment variable
	psScript := fmt.Sprintf(`
$currentPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
$newPath = '%s'
if ($currentPath -notlike "*$newPath*") {
    if ($currentPath) {
        $updatedPath = $newPath + ';' + $currentPath
    } else {
        $updatedPath = $newPath
    }
    [Environment]::SetEnvironmentVariable('PATH', $updatedPath, 'User')
    Write-Host "Successfully added $newPath to user PATH"
} else {
    Write-Host "PATH already contains $newPath"
}`, binDir)

	// Execute the PowerShell script
	execCmd := exec.Command("powershell", "-Command", psScript)
	output, err := execCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update Windows PATH: %v, output: %s", err, string(output))
	}

	fmt.Printf("PATH update output: %s\n", string(output))
	fmt.Printf("Successfully updated Windows user PATH to include %s\n", binDir)
	fmt.Printf("Note: You may need to restart your terminal for the PATH change to take effect\n")
	return nil
}

/*
updateUnixPath adds the ~/bin directory to PATH in shell configuration files.
It checks common shell configuration files (.zshrc, .bash_profile, .bashrc, .profile)
in order of preference and adds the PATH export statement if not already present.

Parameters:
  - homeDir: The user's home directory path

Returns:
  - error: Any error that occurred during the PATH update process
*/
func updateUnixPath(homeDir string) error {
	pathExport := `export PATH="$HOME/bin:$PATH"`

	/* List of shell configuration files to update (in order of preference) */
	configFiles := []string{".zshrc", ".bash_profile", ".bashrc", ".profile"}

	/* Check if PATH export already exists in any file */
	for _, configFile := range configFiles {
		configPath := filepath.Join(homeDir, configFile)
		if pathAlreadyExists(configPath, pathExport) {
			return nil /* Already exists */
		}
	}

	/* Try to update each configuration file in order of preference */
	for _, configFile := range configFiles {
		configPath := filepath.Join(homeDir, configFile)

		/* Skip .zshrc if it doesn't exist (only update existing files) */
		if configFile == ".zshrc" && !fileExists(configPath) {
			continue
		}

		if err := appendToFile(configPath, pathExport); err == nil {
			fmt.Printf("Updated %s with PATH export\n", configFile)
			return nil
		}
	}

	return fmt.Errorf("failed to update any shell configuration file")
}

/*
pathAlreadyExists checks if the PATH export already exists in the given file.
It reads the file line by line and searches for the PATH export statement.

Parameters:
  - filePath: Path to the shell configuration file to check
  - pathExport: The PATH export statement to search for

Returns:
  - bool: true if the PATH export already exists, false otherwise
*/
func pathAlreadyExists(filePath, pathExport string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), pathExport) {
			return true
		}
	}
	return false
}

/*
fileExists checks if a file exists at the given path.
It uses os.Stat to check for file existence and properly handles
the case where the file doesn't exist (os.IsNotExist).

Parameters:
  - filename: Path to the file to check

Returns:
  - bool: true if the file exists, false otherwise
*/
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

/*
appendToFile appends content to a file, creating it if it doesn't exist.
The file is opened in append mode with create flag, and the content is
written with newlines before and after for proper formatting.

Parameters:
  - filePath: Path to the file to append to
  - content: Content to append to the file

Returns:
  - error: Any error that occurred during file operations
*/
func appendToFile(filePath, content string) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, configFileMode)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", filePath, err)
	}
	defer file.Close()

	if _, err := file.WriteString("\n" + content + "\n"); err != nil {
		return fmt.Errorf("failed to write to %s: %w", filePath, err)
	}

	return nil
}

/*
main is the entry point of the Ollama installer.
It orchestrates the entire installation process by:
 1. Fetching the latest Ollama version from GitHub
 2. Constructing the download URL
 3. Installing Ollama to ~/bin/ollama
 4. Updating the user's shell configuration

The program exits with status code 1 if any step fails.
*/
func main() {
	ctx := context.Background()

	fmt.Printf("Detected platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	version, err := getLatestOllamaVersion(ctx)
	if err != nil {
		fmt.Printf("Error getting latest version: %v\n", err)
		os.Exit(1)
	}

	url := getDownloadURL(version)
	fmt.Printf("Latest Ollama version: %s\n", version)
	fmt.Printf("Download URL: %s\n", url)

	if err := installOllama(ctx, url); err != nil {
		fmt.Printf("Installation failed: %v\n", err)
		os.Exit(1)
	}
}
