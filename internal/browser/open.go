package browser

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Open launches url in the default browser. It returns an error only when the
// launch command itself fails; a missing browser is silently ignored because
// the caller always prints a fallback URL.
func Open(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		if isWSL() {
			// wslview (from wslu) is the cleanest option; fall back to cmd.exe.
			if _, err := exec.LookPath("wslview"); err == nil {
				return exec.Command("wslview", url).Start()
			}
			return exec.Command("cmd.exe", "/c", "start", "", url).Start()
		}
		return exec.Command("xdg-open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}
