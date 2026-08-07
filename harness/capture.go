package main

import (
	"path/filepath"
	"regexp"
	"strings"
)

// straceSyscalls is the -e trace= filter passed to strace. Keep in sync with the
// syscalls ParseStrace knows how to interpret.
const straceSyscalls = "openat,open,stat,lstat,newfstatat,access,faccessat,readlink,readlinkat,unlink,unlinkat,execve,read,write,connect,bind"

// pathSyscalls are syscalls whose (first) string argument is a filesystem path.
var pathSyscalls = map[string]bool{
	"open": true, "openat": true, "stat": true, "lstat": true,
	"newfstatat": true, "access": true, "faccessat": true,
	"readlink": true, "readlinkat": true, "unlink": true,
	"unlinkat": true, "execve": true,
}

var (
	// Matches "syscall(args..." with an optional leading pid (strace -f).
	reStraceLine = regexp.MustCompile(`^(?:\[[^\]]*\]\s*)?(?:\d+\s+)?([a-zA-Z_][a-zA-Z0-9_]*)\((.*)`)
	// First double-quoted string in the argument list.
	reFirstString = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)
	// First integer argument (used for read/write fd).
	reFirstInt = regexp.MustCompile(`^\s*(\d+)`)
	// Network address forms.
	reInetAddr = regexp.MustCompile(`inet_addr\("([^"]+)"\)`)
	reHtons    = regexp.MustCompile(`htons\((\d+)\)`)
	reSunPath  = regexp.MustCompile(`sun_path="([^"]+)"`)
)

// ParseStrace converts a raw strace log into a normalized list of
// "syscall:target" entries. Path syscalls yield "openat:/etc/passwd"; network
// syscalls yield "connect:1.2.3.4:80"; read/write yield "read:fd=3" (there is no
// well-defined path for an fd-based syscall, so we record the fd for evidence).
func ParseStrace(log string) []string {
	var out []string
	for _, line := range strings.Split(log, "\n") {
		m := reStraceLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		sys, rest := m[1], m[2]
		switch {
		case pathSyscalls[sys]:
			if s := reFirstString.FindStringSubmatch(rest); s != nil {
				out = append(out, sys+":"+unescape(s[1]))
			}
		case sys == "connect" || sys == "bind":
			if addr := parseSockaddr(rest); addr != "" {
				out = append(out, sys+":"+addr)
			}
		case sys == "read" || sys == "write":
			if i := reFirstInt.FindStringSubmatch(rest); i != nil {
				out = append(out, sys+":fd="+i[1])
			}
		}
	}
	return out
}

// parseSockaddr extracts a human-readable address from a connect/bind arg list.
func parseSockaddr(rest string) string {
	if a := reInetAddr.FindStringSubmatch(rest); a != nil {
		addr := a[1]
		if p := reHtons.FindStringSubmatch(rest); p != nil {
			addr += ":" + p[1]
		}
		return addr
	}
	if a := reSunPath.FindStringSubmatch(rest); a != nil {
		return a[1]
	}
	return ""
}

// unescape resolves the two escape sequences strace emits in quoted strings.
func unescape(s string) string {
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

// DetectFilesystemEscapes checks parsed syscalls against the filesystem canaries
// and returns the matches plus whether any escape occurred.
func DetectFilesystemEscapes(syscalls []string, canaries []Canary) ([]FSAccess, bool) {
	var accesses []FSAccess
	escaped := false
	for _, sc := range syscalls {
		sys, target, ok := splitEntry(sc)
		if !ok || !pathSyscalls[sys] {
			continue
		}
		for _, c := range canaries {
			if c.Category != CategoryFilesystem {
				continue
			}
			if pathMatchesCanary(target, c.Path) {
				accesses = append(accesses, FSAccess{
					Path:     target,
					Canary:   c.Path,
					Syscall:  sys,
					Accessed: true,
				})
				escaped = true
			}
		}
	}
	return accesses, escaped
}

// splitEntry splits a "syscall:target" entry.
func splitEntry(entry string) (sys, target string, ok bool) {
	parts := strings.SplitN(entry, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// pathMatchesCanary reports whether an accessed path is the canary itself or
// lives under it (for directory canaries like /root/.ssh).
func pathMatchesCanary(path, canary string) bool {
	path = filepath.Clean(path)
	canary = filepath.Clean(canary)
	if path == canary {
		return true
	}
	return strings.HasPrefix(path, canary+"/")
}
