package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/gptscript-ai/gptscript/pkg/counter"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type Options struct {
	DumpState     string `usage:"Dump the internal execution state to a file"`
	DebugMessages bool   `usage:"Enable logging of chat completion calls"`
}

func Complete(opts ...Options) (result Options) {
	for _, opt := range opts {
		result.DumpState = types.FirstSet(opt.DumpState, result.DumpState)
		result.DebugMessages = types.FirstSet(opt.DebugMessages, result.DebugMessages)
	}
	return
}

type Console struct {
	dumpState     string
	printMessages bool
	callLock      sync.Mutex
}

var prettyIDCounter int64

func (c *Console) Start(_ context.Context, prg *types.Program, _ []string, input string) (runner.Monitor, error) {
	id := counter.Next()
	mon := newDisplay(c.dumpState, c.printMessages)
	mon.callLock = &c.callLock
	mon.dump.ID = fmt.Sprint(id)
	mon.dump.Program = prg
	mon.dump.Input = input

	log.Fields("runID", mon.dump.ID, "input", input, "program", prg, "type", runner.EventTypeRunStart).Debugf("Run started")
	return mon, nil
}

func (c *Console) Pause() func() {
	c.callLock.Lock()
	return func() {
		c.callLock.Unlock()
	}
}

type display struct {
	dump          dump
	printMessages bool
	livePrinter   *livePrinter
	dumpState     string
	callIDMap     map[string]string
	callLock      *sync.Mutex
	usage         types.Usage
}

type livePrinter struct {
	lastContent    map[string]string
	callIDMap      map[string]string
	activePrinters []string
	toPrint        []string
	needsNewline   bool
}

func (l *livePrinter) end() {
	if l == nil {
		return
	}
	if l.needsNewline {
		_, _ = fmt.Fprintln(os.Stderr)
	}
	l.needsNewline = false
	if len(l.activePrinters) > 0 {
		delete(l.lastContent, l.activePrinters[0])
	}
}

func (l *livePrinter) progressStart(c call) {
	if l == nil {
		return
	}
	if !slices.Contains(l.activePrinters, c.ID) {
		l.activePrinters = append(l.activePrinters, c.ID)
	}
	l.toPrint = slices.DeleteFunc(l.toPrint, func(s string) bool {
		return s == c.ID
	})
}

func (l *livePrinter) progressEnd(c call) {
	if l == nil {
		return
	}
	var result []string
	for i, id := range l.activePrinters {
		if id != c.ID {
			result = append(result, id)
			continue
		}

		if i != 0 {
			if !slices.Contains(l.toPrint, id) {
				l.toPrint = append(l.toPrint, id)
			}
			continue
		}

		for _, toPrintID := range l.toPrint {
			content := l.lastContent[toPrintID]
			delete(l.lastContent, toPrintID)
			if content != "" {
				_, _ = fmt.Fprint(os.Stderr, content)
				if !strings.HasSuffix(content, "\n") {
					_, _ = fmt.Fprintln(os.Stderr)
				}
			}
		}

		l.toPrint = nil
		result = l.activePrinters[1:]
		if len(result) > 0 {
			content := l.lastContent[result[0]]
			if content != "" {
				_, _ = fmt.Fprint(os.Stderr, content)
				l.needsNewline = !strings.HasSuffix(content, "\n")
			}
		}
		break
	}
	l.activePrinters = result
}

func (l *livePrinter) formatContent(event runner.Event, c call) string {
	if event.Content == "" {
		return event.Content
	}
	prefix := fmt.Sprintf("         content  [%s] content | ", l.callIDMap[c.ID])
	var lines []string
	for _, line := range strings.Split(event.Content, "\n") {
		if c.ParentID != "" {
			if len(line) > 100 {
				line = line[:100] + " ..."
			}
		}
		lines = append(lines, prefix+line)
	}
	return strings.Join(lines, "\n")
}

func (l *livePrinter) print(event runner.Event, c call) {
	if l == nil {
		return
	}

	content := l.formatContent(event, c)
	last := l.lastContent[c.ID]
	l.lastContent[c.ID] = content

	if len(l.activePrinters) > 0 && l.activePrinters[0] == c.ID && content != "" {
		line, ok := strings.CutPrefix(content, last)
		if !ok && last != "" {
			_, _ = fmt.Fprintln(os.Stderr)
		}
		if line != "" {
			_, _ = fmt.Fprint(os.Stderr, line)
			l.needsNewline = !strings.HasSuffix(line, "\n")
		}
	}
}

func (d *display) Event(event runner.Event) {
	d.callLock.Lock()
	defer d.callLock.Unlock()

	var (
		currentIndex = -1
		currentCall  call
	)

	for i, existing := range d.dump.Calls {
		if event.CallContext.ID == existing.ID {
			currentIndex = i
			currentCall = existing
			break
		}
	}

	if currentIndex == -1 {
		currentIndex = len(d.dump.Calls)
		currentCall = call{
			ID:       event.CallContext.ID,
			ParentID: event.CallContext.ParentID,
			ToolID:   event.CallContext.Tool.ID,
		}
		d.dump.Calls = append(d.dump.Calls, currentCall)
	}

	log := log.Fields(
		"id", currentCall.ID,
		"parentID", currentCall.ParentID,
		"toolID", currentCall.ToolID,
		"type", event.Type,
	)

	_, ok := d.callIDMap[currentCall.ID]
	if !ok {
		prettyID := fmt.Sprint(atomic.AddInt64(&prettyIDCounter, 1))
		d.callIDMap[currentCall.ID] = prettyID
	}

	callName := callName{
		prettyIDMap:           d.callIDMap,
		call:                  &currentCall,
		prg:                   d.dump.Program,
		calls:                 d.dump.Calls,
		toolCategory:          event.CallContext.ToolCategory,
		userSpecifiedToolName: event.CallContext.ToolName,
	}

	d.usage.PromptTokens += event.Usage.PromptTokens
	d.usage.CompletionTokens += event.Usage.CompletionTokens
	d.usage.TotalTokens += event.Usage.TotalTokens

	switch event.Type {
	case runner.EventTypeCallStart:
		d.livePrinter.progressStart(currentCall)
		d.livePrinter.end()
		currentCall.Start = event.Time
		currentCall.Input = event.Content
		log.Fields("input", event.Content).Infof("started  [%s]", callName)
	case runner.EventTypeCallSubCalls:
		d.livePrinter.progressEnd(currentCall)
	case runner.EventTypeCallProgress:
		d.livePrinter.print(event, currentCall)
	case runner.EventTypeCallContinue:
		d.livePrinter.progressStart(currentCall)
		d.livePrinter.end()
		log.Fields("toolResults", event.ToolResults).Infof("continue [%s]", callName)
	case runner.EventTypeChat:
		d.livePrinter.end()
		if event.ChatRequest == nil {
			log = log.Fields(
				"completionID", event.ChatCompletionID,
				"response", toJSON(event.ChatResponse),
				"cached", event.ChatResponseCached,
			)
		} else {
			log.Infof("sent     [%s]", callName)
			log = log.Fields(
				"completionID", event.ChatCompletionID,
				"request", toJSON(event.ChatRequest),
			)
		}
		if d.printMessages {
			log.Infof("messages")
		} else {
			log.Debugf("debug")
		}
		currentCall.Messages = append(currentCall.Messages, message{
			CompletionID: event.ChatCompletionID,
			Request:      event.ChatRequest,
			Response:     event.ChatResponse,
			Cached:       event.ChatResponseCached,
		})
	case runner.EventTypeCallFinish:
		d.livePrinter.progressEnd(currentCall)
		d.livePrinter.end()
		currentCall.End = event.Time
		currentCall.Output = event.Content
		log.Fields("output", event.Content).Infof("ended    [%s]", callName)
	}

	d.dump.Calls[currentIndex] = currentCall
}

func (d *display) Stop(_ context.Context, output string, err error) {
	d.callLock.Lock()
	defer d.callLock.Unlock()

	log.Fields("runID", d.dump.ID, "output", output, "err", err, "type", runner.EventTypeRunFinish).Debugf("Run stopped")
	if d.usage.TotalTokens > 0 {
		log.Fields("runID", d.dump.ID, "total", d.usage.TotalTokens, "prompt", d.usage.PromptTokens, "completion", d.usage.CompletionTokens).Infof("usage   ")
	}
	d.dump.Output = output
	d.dump.Err = err
	if d.dumpState != "" {
		f, err := os.Create(d.dumpState)
		if err == nil {
			_ = d.Dump(f)
			_ = f.Close()
		}
	}
}

func NewConsole(opts ...Options) *Console {
	opt := Complete(opts...)
	return &Console{
		dumpState:     opt.DumpState,
		printMessages: opt.DebugMessages,
	}
}

func newDisplay(dumpState string, printMessages bool) *display {
	display := &display{
		dumpState:     dumpState,
		callIDMap:     make(map[string]string),
		printMessages: printMessages,
	}
	display.livePrinter = &livePrinter{
		lastContent: map[string]string{},
		callIDMap:   display.callIDMap,
	}
	return display
}

func (d *display) Dump(out io.Writer) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(d.dump)
}

func (d *display) Pause() func() {
	d.callLock.Lock()
	return func() {
		d.callLock.Unlock()
	}
}

func toJSON(obj any) jsonDump {
	return jsonDump{obj: obj}
}

type jsonDump struct {
	obj any
}

func (j jsonDump) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.obj)
}

func (j jsonDump) String() string {
	d, err := json.Marshal(j.obj)
	if err != nil {
		return err.Error()
	}
	return string(d)
}

type callName struct {
	prettyIDMap           map[string]string
	call                  *call
	prg                   *types.Program
	calls                 []call
	toolCategory          engine.ToolCategory
	userSpecifiedToolName string
}

func (c callName) String() string {
	var (
		msg         []string
		currentCall = c.call
	)

	if c.toolCategory != engine.NoCategory {
		// We want to print the credential tool in the same format that the user referenced it, if possible.
		if c.userSpecifiedToolName != "" {
			return fmt.Sprintf("%s: %s", c.toolCategory, color.YellowString(c.userSpecifiedToolName))
		}
		return fmt.Sprintf("%s: %s", c.toolCategory, color.YellowString(currentCall.ToolID))
	}

	for {
		tool := c.prg.ToolSet[currentCall.ToolID]
		name := tool.Parameters.Name
		if name == "" {
			name = tool.Source.Location
		}
		if currentCall.ID != "1" {
			name += "(" + c.prettyIDMap[currentCall.ID] + ")"
		}
		msg = append(msg, name)
		found := false
		for _, parent := range c.calls {
			if parent.ID == currentCall.ParentID {
				found = true
				currentCall = &parent
				break
			}
		}
		if !found {
			break
		}
	}

	slices.Reverse(msg)
	result := strings.Join(msg[1:], "->")
	if result == "" {
		return "main"
	}
	return result
}

type dump struct {
	ID      string         `json:"id,omitempty"`
	Program *types.Program `json:"program,omitempty"`
	Calls   []call         `json:"calls,omitempty"`
	Input   string         `json:"input,omitempty"`
	Output  string         `json:"output,omitempty"`
	Err     error          `json:"err,omitempty"`
}

type message struct {
	CompletionID string `json:"completionID,omitempty"`
	Request      any    `json:"request,omitempty"`
	Response     any    `json:"response,omitempty"`
	Cached       bool   `json:"cached,omitempty"`
}

type call struct {
	ID       string    `json:"id,omitempty"`
	ParentID string    `json:"parentID,omitempty"`
	ToolID   string    `json:"toolID,omitempty"`
	Messages []message `json:"messages,omitempty"`
	Start    time.Time `json:"start,omitempty"`
	End      time.Time `json:"end,omitempty"`
	Input    string    `json:"input,omitempty"`
	Output   string    `json:"output,omitempty"`
}

func (c call) String() string {
	return c.ID
}
