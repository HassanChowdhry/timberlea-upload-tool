# How to run
- 1) Clone the repo
- 2) Run `./ollama-installer`. This will run the included library
- 3) Restart the terminal session

# Dev Mode
## Create binary build script
```bash
GOOS=linux GOARCH=amd64 go build -o ollama-installer main.go
```
