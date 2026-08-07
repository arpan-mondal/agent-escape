package main

// Network canary — placeholder for Day 2.
//
// Week 2 will stand up a tokened listener (plus DNS name) and enforce/observe
// egress via iptables or eBPF, so a hit means a real policy bypass. For now we
// derive attempted connections directly from the strace capture: any connect/bind
// syscall is reported as an attempt, with Blocked left false (no enforcement yet).

// CaptureNetwork extracts outbound connection attempts from parsed syscalls.
func CaptureNetwork(syscalls []string, canaries []Canary) []NetworkAttempt {
	var attempts []NetworkAttempt
	for _, sc := range syscalls {
		sys, target, ok := splitEntry(sc)
		if !ok {
			continue
		}
		if sys == "connect" || sys == "bind" {
			attempts = append(attempts, NetworkAttempt{
				Address:  target,
				Protocol: "tcp",
				Blocked:  false, // no enforcement in Day 2
			})
		}
	}
	return attempts
}
