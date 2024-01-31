package runner

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"

	"github.com/acorn-io/gptscript/pkg/engine"
	"github.com/pterm/pterm"
)

type display struct {
	progress chan Event
	states   []state
	done     chan struct{}
	area     *pterm.AreaPrinter
	quiet    bool
}

func newDisplay(quiet bool) *display {
	return &display{
		quiet: quiet,
	}
}

func (d *display) Start(ctx context.Context) (err error) {
	if !d.quiet {
		d.area, err = pterm.DefaultArea.
			//WithFullscreen(true).
			WithRemoveWhenDone(true).
			Start("Starting...")
		if err != nil {
			return err
		}
	}

	d.progress = make(chan Event)
	d.done = make(chan struct{})
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				d.print()
			case <-d.done:
				return
			}
		}
	}()
	go func() {
		for msg := range d.progress {
			d.addEvent(msg)
		}
		close(d.done)
	}()
	return nil
}

func (d *display) Dump(out io.Writer) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		State []state `json:"state,omitempty"`
	}{
		State: d.states,
	})
}

func (d *display) Stop() error {
	close(d.progress)
	<-d.done
	if d.area != nil {
		if err := d.area.Stop(); err != nil {
			return err
		}
	}

	return nil
}

func splitCount(line string, length int) (head, tail string) {
	if len(line) < length {
		return line, ""
	}
	return line[:length], line[length:]
}

func multiLineWrite(out io.StringWriter, prefix, lines string) {
	if lines == "" {
		_, _ = out.WriteString("\n")
	}
	width := pterm.GetTerminalWidth()
	for _, line := range strings.Split(lines, "\n") {
		line = prefix + line
		for {
			head, tail := splitCount(line, width)
			_, _ = out.WriteString(head)
			_, _ = out.WriteString("\n")

			if tail == "" {
				break
			}
			line = prefix + tail
		}
	}
}

func (d *display) printState(s state, depth int) string {
	if !s.Running {
		return ""
	}

	buf := &strings.Builder{}
	prefix := strings.Repeat("  ", depth)
	inPrefix := prefix + "  |<- "
	outPrefix := prefix + "  |-> "

	buf.WriteString(prefix)
	name := s.Context.Tool.Name
	if name == "" {
		name = "main"
	}
	if s.Running {
		buf.WriteString("(running ")
		buf.WriteString(name)
		buf.WriteString(")\n")
		if s.Input != "" {
			multiLineWrite(buf, inPrefix, "args: "+s.Input)
		}
	} else {
		buf.WriteString("(done ")
		buf.WriteString(name)
		buf.WriteString(") ")
		buf.WriteString("args: ")
		head, tail := splitCount(s.Input, 40)
		buf.WriteString(head)
		if tail != "" {
			buf.WriteString("...")
		}
		buf.WriteString("\n")
	}

	childRunning := false
	for _, state := range d.states {
		if state.Context != nil && state.Context.Parent != nil && state.Context.Parent.ID == s.Context.ID {
			if state.Running {
				childRunning = true
			}
			buf.WriteString(d.printState(state, depth+1))
		}
	}

	if depth == 0 && !childRunning {
		if len(s.Input) > 0 && len(s.Output) > 0 {
			multiLineWrite(buf, outPrefix, "---")
		}

		multiLineWrite(buf, outPrefix, s.Output)
	}

	return buf.String()
}

func (d *display) print() {
	if d.quiet {
		return
	}
	d.area.Update(d.String() + "\n")
}

func (d *display) String() string {
	buf := strings.Builder{}
	if len(d.states) > 0 {
		buf.WriteString(d.printState(d.states[0], 0))
	}

	return buf.String()
}

func (d *display) addEvent(msg Event) {
	found := false
	for i, state := range d.states {
		if state.Context.ID != msg.Context.ID {
			continue
		}
		found = true
		switch msg.Type {
		case EventTypeUpdate:
			state.Output = msg.Content
		case EventTypeStop:
			state.Running = false
			state.Output = msg.Content
			state.End = msg.Time
		}
		d.states[i] = state
	}
	if !found && msg.Type == EventTypeStart {
		d.states = append(d.states, state{
			Context: msg.Context,
			Running: true,
			Start:   msg.Time,
			Input:   msg.Content,
		})
	} else if !found {
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		enc.Encode(msg)
		panic("why?")
	}
}

type state struct {
	Context *engine.Context `json:"context,omitempty"`
	Running bool            `json:"running,omitempty"`
	Start   time.Time       `json:"start,omitempty"`
	End     time.Time       `json:"end,omitempty"`
	Input   string          `json:"input,omitempty"`
	Output  string          `json:"output,omitempty"`
}
