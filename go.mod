module github.com/arpan-mondal/agent-escape

go 1.26

// No external dependencies: the Docker/gVisor backends shell out to the `docker`
// CLI via os/exec, and the CLI uses the stdlib `flag` package. This keeps the
// build offline-clean. Swapping to the official Docker Go SDK is a Week-2 option.
