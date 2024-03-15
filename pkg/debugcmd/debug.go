package debugcmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type WrappedCmd struct {
	c   *exec.Cmd
	r   recorder
	Env []string
	Dir string
}

func (w *WrappedCmd) Run() error {
	if len(w.Env) > 0 {
		w.c.Env = w.Env
	}
	if w.Dir != "" {
		w.c.Dir = w.Dir
	}
	if err := w.c.Run(); err != nil {
		msg := w.r.dump()
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

func New(ctx context.Context, arg string, args ...string) *WrappedCmd {
	w := &WrappedCmd{
		c: exec.CommandContext(ctx, arg, args...),
	}
	setupDebug(w)
	return w
}

type entry struct {
	err  bool
	data []byte
}

type recorder struct {
	lock    sync.Mutex
	entries []entry
}

func (r *recorder) dump() string {
	var errMessage strings.Builder
	for _, entry := range r.entries {
		if entry.err {
			errMessage.Write(entry.data)
			_, _ = os.Stderr.Write(entry.data)
		} else {
			_, _ = os.Stdout.Write(entry.data)
		}
	}
	return errMessage.String()
}

type writer struct {
	err bool
	r   *recorder
}

func (w *writer) Write(data []byte) (int, error) {
	w.r.lock.Lock()
	defer w.r.lock.Unlock()

	cp := make([]byte, len(data))
	copy(cp, data)

	w.r.entries = append(w.r.entries, entry{
		err:  w.err,
		data: cp,
	})

	return len(data), nil
}

func setupDebug(w *WrappedCmd) {
	if log.IsDebug() {
		w.c.Stdout = os.Stdout
		w.c.Stderr = os.Stderr
	} else {
		w.c.Stdout = &writer{
			r: &w.r,
		}
		w.c.Stderr = &writer{
			err: true,
			r:   &w.r,
		}
	}
}
