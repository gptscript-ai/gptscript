package monitor

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type Event struct {
	runner.Event `json:",inline"`
	Program      *types.Program `json:"program,omitempty"`
	Input        string         `json:"input,omitempty"`
	Output       string         `json:"output,omitempty"`
	Err          string         `json:"err,omitempty"`
}

type fileFactory struct {
	file *os.File
}

// NewFileFactory creates a new monitor factory that writes events to the location specified.
// The location can be one of three things:
// 1. a file descriptor/handle in the form "fd://2"
// 2. a file name
// 3. a named pipe in the form "\\.\pipe\my-pipe"
func NewFileFactory(loc string) (runner.MonitorFactory, error) {
	var (
		file *os.File
		err  error
	)

	if strings.HasPrefix(loc, "fd://") {
		fd, err := strconv.Atoi(strings.TrimPrefix(loc, "fd://"))
		if err != nil {
			return nil, err
		}

		file = os.NewFile(uintptr(fd), "events")
	} else {
		file, err = os.OpenFile(loc, os.O_WRONLY|os.O_CREATE, 0)
		if err != nil {
			return nil, err
		}
	}

	return &fileFactory{
		file: file,
	}, nil
}

func (s fileFactory) Start(_ context.Context, prg *types.Program, env []string, input string) (runner.Monitor, error) {
	fd := &fd{
		prj:   prg,
		env:   env,
		input: input,
		file:  s.file,
	}

	fd.event(Event{
		Event: runner.Event{
			Time: time.Now(),
			Type: runner.EventTypeRunStart,
		},
		Program: prg,
	})

	return fd, nil
}

type fd struct {
	prj     *types.Program
	env     []string
	input   string
	file    *os.File
	runLock sync.Mutex
}

func (f *fd) Event(event runner.Event) {
	f.event(Event{
		Event: event,
		Input: f.input,
	})
}

func (f *fd) event(event Event) {
	f.runLock.Lock()
	defer f.runLock.Unlock()
	b, err := json.Marshal(event)
	if err != nil {
		log.Errorf("Failed to marshal event: %v", err)
		return
	}

	if _, err = f.file.Write(append(b, '\n', '\n')); err != nil {
		log.Errorf("Failed to write event to file: %v", err)
	}
}

func (f *fd) Stop(output string, err error) {
	e := Event{
		Event: runner.Event{
			Time: time.Now(),
			Type: runner.EventTypeRunFinish,
		},
		Input:  f.input,
		Output: output,
	}
	if err != nil {
		e.Err = err.Error()
	}

	f.event(e)
	if err = f.file.Close(); err != nil {
		log.Errorf("Failed to close file: %v", err)
	}
}

func (f *fd) Pause() func() {
	f.runLock.Lock()
	return func() {
		f.runLock.Unlock()
	}
}
