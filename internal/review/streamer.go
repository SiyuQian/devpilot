package review

import (
	"fmt"
	"io"
	"os"

	"github.com/siyuqian/devpilot/internal/executor"
)

// reviewStreamer handles real-time display of Claude review events.
// Text content is streamed to stdout; progress indicators go to stderr.
type reviewStreamer struct {
	out     io.Writer // text output (stdout)
	err     io.Writer // progress indicators (stderr)
	showTTY bool      // whether to show progress on stderr
}

// newReviewStreamer creates a streamer that writes text to stdout and
// progress to stderr. Progress indicators are suppressed when stdout
// is not a TTY.
func newReviewStreamer() *reviewStreamer {
	isTTY := false
	if fi, err := os.Stdout.Stat(); err == nil {
		isTTY = (fi.Mode() & os.ModeCharDevice) != 0
	}
	return &reviewStreamer{
		out:     os.Stdout,
		err:     os.Stderr,
		showTTY: isTTY,
	}
}

// HandleEvent processes a single Claude stream-json event.
func (s *reviewStreamer) HandleEvent(event executor.ClaudeEvent) {
	switch e := event.(type) {
	case executor.ClaudeAssistantMsg:
		for _, block := range e.Content {
			switch b := block.(type) {
			case executor.TextBlock:
				_, _ = fmt.Fprint(s.out, b.Text)
			case executor.ToolUseBlock:
				if s.showTTY {
					_, _ = fmt.Fprintf(s.err, "[tool] %s\n", b.Name)
				}
			}
		}
	case executor.ClaudeResultMsg:
		// Print a trailing newline after all text output
		_, _ = fmt.Fprintln(s.out)
	}
}
