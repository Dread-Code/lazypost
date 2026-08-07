package clipboard

import (
	"bytes"
	"errors"
	"os/exec"
	"runtime"
)

// ErrNoTool is returned when no platform clipboard command is available.
var ErrNoTool = errors.New("no clipboard tool available")

// Write puts text on the system clipboard via the platform tool
// (pbcopy, wl-copy/xclip/xsel, clip). Returns an error if none runs.
func Write(text string) error {
	for _, args := range candidates() {
		path, err := exec.LookPath(args[0])
		if err != nil {
			continue
		}
		cmd := exec.Command(path, args[1:]...)
		cmd.Stdin = bytes.NewBufferString(text)
		if out, err := cmd.CombinedOutput(); err != nil {
			continue
		} else if len(out) > 0 {
			// tools may log; treat as failure
			continue
		}
		return nil
	}
	return ErrNoTool
}

func candidates() [][]string {
	switch runtime.GOOS {
	case "darwin":
		return [][]string{{"pbcopy"}}
	case "linux":
		return [][]string{
			{"wl-copy"},
			{"xclip", "-selection", "clipboard"},
			{"xsel", "--clipboard", "--input"},
		}
	case "windows":
		return [][]string{{"clip"}}
	}
	return nil
}
