package clipboard

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"runtime"
)

// ErrNoTool is returned when no platform clipboard command is available.
var ErrNoTool = errors.New("no clipboard tool available")

// Write puts text on the system clipboard via the platform tool
// (pbcopy, wl-copy/xclip/xsel, clip). Returns an error if none runs.
func Write(text string) error {
	return WriteContext(context.Background(), text)
}

// WriteContext puts text on the system clipboard with cancellation.
func WriteContext(ctx context.Context, text string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for _, args := range candidates() {
		path, err := exec.LookPath(args[0])
		if err != nil {
			continue
		}
		cmd := exec.CommandContext(ctx, path, args[1:]...)
		cmd.Stdin = bytes.NewBufferString(text)
		if out, err := cmd.CombinedOutput(); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			continue
		} else if len(out) > 0 {
			// tools may log; treat as failure
			continue
		}
		return nil
	}
	return ErrNoTool
}

// candidates lists clipboard commands per OS, in preference order; the
// first one that exists and runs successfully wins.
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
