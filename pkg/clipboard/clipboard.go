package clipboard

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var ErrUnavailable = errors.New("no clipboard command found")

type command struct {
	name string
	args []string
}

func Copy(text string) error {
	for _, c := range commands() {
		path, err := exec.LookPath(c.name)
		if err != nil {
			continue
		}
		cmd := exec.Command(path, c.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return ErrUnavailable
}

func commands() []command {
	switch runtime.GOOS {
	case "darwin":
		return []command{{name: "pbcopy"}}
	case "windows":
		return []command{{name: "clip"}}
	}
	var out []command
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		out = append(out, command{name: "wl-copy"})
	}
	return append(out,
		command{name: "xclip", args: []string{"-selection", "clipboard"}},
		command{name: "xsel", args: []string{"--clipboard", "--input"}},
		command{name: "clip.exe"},
	)
}
