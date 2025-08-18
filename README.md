# How to run
1. Clone the repo
2. Build the appropriate binary for your platform (see build instructions below)
3. Run the binary:
   - **Windows**: `ollama-installer.exe`
   - **Linux/macOS**: `./ollama-installer`

## Installation Locations
- **Windows**: Installs to `%USERPROFILE%\AppData\Local\Programs\Ollama\ollama.exe` (standard Windows location)
- **Linux/macOS**: Installs to `~/bin/ollama` and automatically updates your PATH

# Building the Binary

## Cross-Platform Build Commands
### Windows Binary
```cmd
go build -o ollama-installer.exe main.go
```

### Linux/macOS Binary
```bash
go build -o ollama-installer main.go
```

## Where are the binaries created?

The binaries are created in the same directory where you run the build command (your project root directory). After building, you should see:

- `ollama-installer.exe` (for Windows)
- `ollama-installer` (for Linux/macOS)

**To check if the build worked:**

Windows:
```cmd
dir ollama-installer.exe
```

Linux/macOS:
```bash
ls -la ollama-installer
```

# Using Ollama

After installation, you can use Ollama with these commands:

## Basic Usage

1. **Start Ollama service** (run in background):
   ```bash
   ollama serve
   ```

2. **In a new terminal, run a model**:
   ```bash
   # Download and run a model (e.g., Llama 2)
   ollama run llama2
   
   # Or try a smaller model
   ollama run llama2:7b
   ```

3. **List available models**:
   ```bash
   ollama list
   ```

4. **Pull a model without running**:
   ```bash
   ollama pull codellama
   ```

5. **Check version**:
   ```bash
   ollama --version
   ```

## Getting Started
1. First run `ollama serve` to start the Ollama service
2. Open a new terminal window
3. Run `ollama run llama2` to download and start chatting with Llama 2
4. Type your questions and press Enter to chat

# Uninstalling Ollama

## Windows Uninstall

To completely remove Ollama installed by this tool:

1. **Stop Ollama service** (if running):
   ```cmd
   taskkill /F /IM ollama.exe
   ```

2. **Delete the Ollama directory**:
   ```cmd
   rmdir /S /Q "%USERPROFILE%\AppData\Local\Programs\Ollama"
   ```

3. **Remove from PATH** (if manually added):
   ```cmd
   # View current PATH to verify
   echo %PATH%
   
   # If you manually added it, remove via System Properties > Environment Variables
   # Or use PowerShell to remove it:
   powershell -Command "$currentPath = [Environment]::GetEnvironmentVariable('PATH', 'User'); $newPath = $currentPath -replace '%USERPROFILE%\\AppData\\Local\\Programs\\Ollama;?', ''; [Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')"
   ```

4. **Remove models and data** (optional):
   ```cmd
   rmdir /S /Q "%USERPROFILE%\.ollama"
   ```

## Linux/macOS Uninstall

To completely remove Ollama installed by this tool:

1. **Stop Ollama service** (if running):
   ```bash
   pkill ollama
   ```

2. **Remove the binary**:
   ```bash
   rm ~/bin/ollama
   ```

3. **Remove from shell configuration**:
   ```bash
   # For Zsh users
   sed -i '/export PATH="\$HOME\/bin:\$PATH"/d' ~/.zshrc
   
   # For Bash users
   sed -i '/export PATH="\$HOME\/bin:\$PATH"/d' ~/.bashrc
   sed -i '/export PATH="\$HOME\/bin:\$PATH"/d' ~/.bash_profile
   ```

4. **Remove models and data** (optional):
   ```bash
   rm -rf ~/.ollama
   ```

5. **Reload shell configuration**:
   ```bash
   source ~/.zshrc  # or ~/.bashrc
   ```


## Troubleshooting

### Windows

**If Ollama isn't found after installation:**

1. **Check if it was installed:**
   ```cmd
   dir "%USERPROFILE%\AppData\Local\Programs\Ollama\ollama.exe"
   ```

2. **Test if it worked:**
   ```cmd
   ollama --version
   ```

3. **If still not working, run directly:**
   ```cmd
   "%USERPROFILE%\AppData\Local\Programs\Ollama\ollama.exe" --version
   ```

4. **If the directory is not in PATH, add it manually:**
   
   **Option A: Temporary (current terminal only):**
   ```cmd
   set PATH=%USERPROFILE%\AppData\Local\Programs\Ollama;%PATH%
   ```

   **Option B: Permanent (all future terminals):**
   ```cmd
   setx PATH "%USERPROFILE%\AppData\Local\Programs\Ollama;%PATH%"
   ```
   *Note: Close and reopen your terminal after using setx*

### Linux/macOS

**If Ollama isn't found after installation:**

1. **Check if it was installed:**
   ```bash
   ls -la ~/bin/ollama
   ```

2. **Test if it worked (restart terminal first):**
   ```bash
   ollama --version
   ```

3. **If still not working, run directly:**
   ```bash
   ~/bin/ollama --version
   ```

4. **Manually add to PATH if needed:**
   ```bash
   export PATH="$HOME/bin:$PATH"
   ```
   
   To make it permanent, add the above line to your shell config file:
   - **Zsh**: `~/.zshrc`
   - **Bash**: `~/.bashrc` or `~/.bash_profile`

### General Notes
- Always restart your terminal after installation for PATH changes to take effect
- On Windows, the installer uses the standard Ollama installation directory which should already be in your PATH
