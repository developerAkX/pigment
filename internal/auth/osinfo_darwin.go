package auth

import (
	"os/exec"
	"runtime"
	"strings"
)

func formatOSInfoImpl() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}

	ver := "26.0.1" // fallback
	if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
		ver = strings.TrimSpace(string(out))
	}

	return "Mac OS " + ver + "; " + arch
}
