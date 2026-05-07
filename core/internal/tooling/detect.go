package tooling

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func currentRuntime() RuntimeInfo {
	homeDir, _ := os.UserHomeDir()
	return RuntimeInfo{
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		IsWSL:   isWSL(),
		HomeDir: homeDir,
	}
}

func resolveExecutable(command string) string {
	lookupCommand := "which"
	if runtime.GOOS == "windows" {
		lookupCommand = "where"
	}

	output, err := exec.Command(lookupCommand, command).Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func isWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	content, err := os.ReadFile("/proc/version")
	if err == nil && strings.Contains(strings.ToLower(string(content)), "microsoft") {
		return true
	}

	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}

	return false
}
