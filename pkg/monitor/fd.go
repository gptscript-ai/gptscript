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
	fileName     string
	file         *os.File
	lock         sync.Mutex
	runningCount int
}

// NewFileFactory creates a new monitor factory that writes events to the location specified.
// The location can be one of three things:
// 1. a file descriptor/handle in the form "fd://2"
// 2. a file name
// 3. a named pipe in the form "\\.\pipe\my-pipe"
func NewFileFactory(loc string) (runner.MonitorFactory, error) {
	return &fileFactory{
		fileName: loc,
	}, nil
}

func (s *fileFactory) Start(_ context.Context, prg *types.Program, env []string, input string) (runner.Monitor, error) {
	s.lock.Lock()
	s.runningCount++
	if s.runningCount == 1 {
		if err := s.openFile(); err != nil {
			s.runningCount--
			s.lock.Unlock()
			return nil, err
		}
	}
	s.lock.Unlock()

	fd := &fd{
		prj:     prg,
		env:     env,
		input:   input,
		file:    s.file,
		factory: s,
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

func (s *fileFactory) Pause() func() {
	return func() {}
}

func (s *fileFactory) close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.runningCount--
	if s.runningCount == 0 {
		if err := s.file.Close(); err != nil {
			log.Errorf("error closing monitor file: %v", err)
		}
	}
}

func (s *fileFactory) openFile() error {
	var (
		err  error
		file *os.File
	)
	if strings.HasPrefix(s.fileName, "fd://") {
		fd, err := strconv.Atoi(strings.TrimPrefix(s.fileName, "fd://"))
		if err != nil {
			return err
		}

		file = os.NewFile(uintptr(fd), "events")
	} else {
		file, err = os.OpenFile(s.fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
	}

	s.file = file
	return nil
}

type fd struct {
	prj     *types.Program
	env     []string
	input   string
	file    *os.File
	runLock sync.Mutex
	factory *fileFactory
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

func (f *fd) Stop(_ context.Context, output string, err error) {
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
	f.factory.close()
}

func (f *fd) Pause() func() {
	f.runLock.Lock()
	return func() {
		f.runLock.Unlock()
	}
}
