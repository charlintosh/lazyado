package clipboard

import (
	"os/exec"
	"runtime"
	"strings"
)

// Copy copies the given text to the system clipboard
func Copy(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "echo|set /p="+text+"|clip")
	case "darwin":
		cmd = exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
	default: // linux and others
		// Try xclip first, fall back to xsel
		cmd = exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
	}

	return cmd.Run()
}
