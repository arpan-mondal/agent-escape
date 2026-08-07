package main

import "fmt"

func init() {
	registerBackend(gvisorBackend{})
}

// gvisorBackend runs containers under gVisor's userspace kernel (the Sentry).
//
// gVisor is "Docker + a runtime flag": the interface is identical to the Docker
// backend, only the isolation model differs. We therefore reuse dockerSandbox and
// just set the runtime to "runsc" (which must be registered with Docker via
// `runsc install` on a Linux host — see notes/interface.md).
type gvisorBackend struct{}

func (gvisorBackend) Name() string { return "gvisor" }

func (gvisorBackend) New(cfg SandboxConfig) (Sandbox, error) {
	if cfg.Runtime == "" {
		cfg.Runtime = "runsc"
	}
	if cfg.Runtime != "runsc" {
		return nil, fmt.Errorf("gvisor backend requires runtime 'runsc', got %q", cfg.Runtime)
	}
	return newDockerSandbox(cfg), nil
}
