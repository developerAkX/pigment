package auth

import "runtime"

func formatOSInfoImpl() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}
	return "Linux; " + arch
}
