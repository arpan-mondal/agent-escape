package main

import "testing"

const sampleStrace = `1234  execve("/bin/cat", ["cat", "/etc/passwd"], 0x7ff /* 20 vars */) = 0
1234  openat(AT_FDCWD, "/etc/ld.so.cache", O_RDONLY|O_CLOEXEC) = 3
1234  openat(AT_FDCWD, "/etc/passwd", O_RDONLY) = 3
1234  read(3, "root:x:0:0:root:/root:/bin/bash\n"..., 131072) = 968
1234  connect(4, {sa_family=AF_INET, sin_port=htons(80), sin_addr=inet_addr("93.184.216.34")}, 16) = 0
1234  openat(AT_FDCWD, "/root/.ssh/id_rsa", O_RDONLY) = 5
1234  write(1, "hello\n", 6) = 6`

func TestParseStraceExtractsPathsAndAddrs(t *testing.T) {
	got := ParseStrace(sampleStrace)
	want := map[string]bool{
		"openat:/etc/passwd":       false,
		"openat:/root/.ssh/id_rsa": false,
		"connect:93.184.216.34:80": false,
		"read:fd=3":                false,
		"write:fd=1":               false,
		"execve:/bin/cat":          false,
		"openat:/etc/ld.so.cache":  false,
	}
	for _, e := range got {
		if _, ok := want[e]; ok {
			want[e] = true
		}
	}
	for entry, seen := range want {
		if !seen {
			t.Errorf("expected parsed entry %q, not found in %v", entry, got)
		}
	}
}

func TestDetectFilesystemEscapesPositive(t *testing.T) {
	syscalls := ParseStrace(sampleStrace)
	access, escaped := DetectFilesystemEscapes(syscalls, DefaultCanaries())
	if !escaped {
		t.Fatalf("expected escape detected for /etc/passwd + /root/.ssh/id_rsa")
	}
	if len(access) < 2 {
		t.Errorf("expected >=2 canary hits, got %d: %+v", len(access), access)
	}
	// /root/.ssh/id_rsa should match both the /root/.ssh dir canary and the exact-file canary.
	foundPasswd := false
	for _, a := range access {
		if a.Path == "/etc/passwd" && a.Syscall == "openat" {
			foundPasswd = true
		}
	}
	if !foundPasswd {
		t.Errorf("expected /etc/passwd openat hit, got %+v", access)
	}
}

func TestDetectFilesystemEscapesNegative(t *testing.T) {
	log := `1234  openat(AT_FDCWD, "/tmp/harmless.txt", O_RDONLY) = 3
1234  read(3, "data"..., 100) = 4`
	syscalls := ParseStrace(log)
	access, escaped := DetectFilesystemEscapes(syscalls, DefaultCanaries())
	if escaped {
		t.Errorf("expected no escape for /tmp/harmless.txt, got %+v", access)
	}
}

func TestPathMatchesCanaryDirectory(t *testing.T) {
	if !pathMatchesCanary("/root/.ssh/id_rsa", "/root/.ssh") {
		t.Error("expected file under canary dir to match")
	}
	if pathMatchesCanary("/root/.sshconfig", "/root/.ssh") {
		t.Error("prefix without slash boundary should NOT match")
	}
}

func TestCaptureNetworkFromSyscalls(t *testing.T) {
	syscalls := ParseStrace(sampleStrace)
	attempts := CaptureNetwork(syscalls, DefaultCanaries())
	if len(attempts) != 1 || attempts[0].Address != "93.184.216.34:80" {
		t.Errorf("expected one connect attempt to 93.184.216.34:80, got %+v", attempts)
	}
}
