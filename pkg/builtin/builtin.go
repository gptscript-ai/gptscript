package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/locker"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/prompt"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/jaytaylor/html2text"
)

var SafeTools = map[string]struct{}{
	"sys.abort":        {},
	"sys.chat.finish":  {},
	"sys.chat.history": {},
	"sys.chat.current": {},
	"sys.echo":         {},
	"sys.prompt":       {},
	"sys.time.now":     {},
	"sys.context":      {},
}

var tools = map[string]types.Tool{
	"sys.time.now": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Returns the current date and time in RFC3339 format",
			},
			BuiltinFunc: SysTimeNow,
		},
	},
	"sys.ls": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Lists the contents of a directory",
				Arguments: types.ObjectSchema(
					"dir", "The directory to list"),
			},
			BuiltinFunc: SysLs,
		},
	},
	"sys.read": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Reads the contents of a file",
				Arguments: types.ObjectSchema(
					"filename", "The name of the file to read"),
			},
			BuiltinFunc: SysRead,
		},
	},
	"sys.write": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Write the contents to a file",
				Arguments: types.ObjectSchema(
					"filename", "The name of the file to write to",
					"content", "The content to write"),
			},
			BuiltinFunc: SysWrite,
		},
	},
	"sys.append": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Appends the contents to a file",
				Arguments: types.ObjectSchema(
					"filename", "The name of the file to append to",
					"content", "The content to append"),
			},
			BuiltinFunc: SysAppend,
		},
	},
	"sys.http.get": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Download the contents of a http or https URL",
				Arguments: types.ObjectSchema(
					"url", "The URL to download"),
			},
			BuiltinFunc: SysHTTPGet,
		},
	},
	"sys.http.html2text": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Download the contents of a http or https URL returning the content as rendered text converted from HTML",
				Arguments: types.ObjectSchema(
					"url", "The URL to download"),
			},
			BuiltinFunc: SysHTTPHtml2Text,
		},
	},
	"sys.abort": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Aborts execution",
				Arguments: types.ObjectSchema(
					"message", "The description of the error or unexpected result that caused abort to be called",
				),
			},
			BuiltinFunc: SysAbort,
		},
	},
	"sys.chat.finish": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Concludes the conversation. This can not be used to ask a question.",
				Arguments: types.ObjectSchema(
					"return", "The instructed value to return or a summary of the dialog if no value is instructed",
				),
			},
			BuiltinFunc: SysChatFinish,
		},
	},
	"sys.http.post": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Write contents to a http or https URL using the POST method",
				Arguments: types.ObjectSchema(
					"url", "The URL to POST to",
					"content", "The content to POST",
					"contentType", "The \"content type\" of the content such as application/json or text/plain"),
			},
			BuiltinFunc: SysHTTPPost,
		},
	},
	"sys.find": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Traverse a directory looking for files that match a pattern in the style of the unix find command",
				Arguments: types.ObjectSchema(
					"pattern", "The file pattern to look for. The pattern is a traditional unix glob format with * matching any character and ? matching a single character",
					"directory", "The directory to search in. The current directory \".\" will be used as the default if no argument is passed",
				),
			},
			BuiltinFunc: SysFind,
		},
	},
	"sys.exec": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Execute a command and get the output of the command",
				Arguments: types.ObjectSchema(
					"command", "The command to run including all applicable arguments",
					"directory", "The directory to use as the current working directory of the command. The current directory \".\" will be used if no argument is passed",
				),
			},
			BuiltinFunc: SysExec,
		},
	},
	"sys.getenv": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Gets the value of an OS environment variable",
				Arguments: types.ObjectSchema(
					"name", "The environment variable name to lookup"),
			},
			BuiltinFunc: SysGetenv,
		},
	},
	"sys.download": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Downloads a URL, saving the contents to disk at a given location",
				Arguments: types.ObjectSchema(
					"url", "The URL to download, either http or https.",
					"location", "(optional) The on disk location to store the file. If no location is specified a temp location will be used. If the target file already exists it will fail unless override is set to true.",
					"override", "If true and a file at the location exists, the file will be overwritten, otherwise fail. Default is false"),
			},
			BuiltinFunc: SysDownload,
		},
	},
	"sys.remove": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Removes the specified files",
				Arguments: types.ObjectSchema(
					"location", "The file to remove"),
			},
			BuiltinFunc: SysRemove,
		},
	},
	"sys.stat": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Gets size, modfied time, and mode of the specified file",
				Arguments: types.ObjectSchema(
					"filepath", "The complete path and filename of the file",
				),
			},
			BuiltinFunc: SysStat,
		},
	},
	"sys.prompt": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Prompts the user for input",
				Arguments: types.ObjectSchema(
					"message", "The message to display to the user",
					"fields", "A comma-separated list of fields to prompt for",
					"sensitive", "(true or false) Whether the input should be hidden",
				),
			},
			BuiltinFunc: prompt.SysPrompt,
		},
	},
	"sys.chat.history": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Retrieves the previous chat dialog",
				Arguments:   types.ObjectSchema(),
			},
			BuiltinFunc: SysChatHistory,
		},
	},
	"sys.chat.current": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Retrieves the current chat dialog",
				Arguments:   types.ObjectSchema(),
			},
			BuiltinFunc: SysChatCurrent,
		},
	},
	"sys.context": {
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: "Retrieves the current internal GPTScript tool call context information",
				Arguments:   types.ObjectSchema(),
			},
			BuiltinFunc: SysContext,
		},
	},
}

func ListTools() (result []types.Tool) {
	var keys []string
	for k := range tools {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, key := range keys {
		t, _ := Builtin(key)
		result = append(result, t)
	}

	return
}

func Builtin(name string) (types.Tool, bool) {
	// Legacy syntax not used anymore
	name = strings.TrimSuffix(name, "?")
	t, ok := tools[name]
	t.Parameters.Name = name
	t.ID = name
	t.Instructions = "#!" + name
	return SetDefaults(t), ok
}

func SysFind(_ context.Context, _ []string, input string, _ chan<- string) (string, error) {
	var result []string
	var params struct {
		Pattern   string `json:"pattern,omitempty"`
		Directory string `json:"directory,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return invalidArgument(input, err), nil
	}

	if params.Directory == "" {
		params.Directory = "."
	}

	log.Debugf("Finding files %s in %s", params.Pattern, params.Directory)
	err := fs.WalkDir(os.DirFS(params.Directory), ".", func(pathname string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ok, err := filepath.Match(params.Pattern, d.Name()); err != nil {
			return err
		} else if ok {
			path := filepath.Join(params.Directory, pathname)
			if d.IsDir() {
				path += "/"
			}
			result = append(result, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Sprintf("Failed to traverse directory %s: %v", params.Directory, err.Error()), nil
	}
	if len(result) == 0 {
		return "No files found", nil
	}

	sort.Strings(result)
	return strings.Join(result, "\n"), nil
}

func SysExec(_ context.Context, env []string, input string, progress chan<- string) (string, error) {
	var params struct {
		Command   string `json:"command,omitempty"`
		Directory string `json:"directory,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return invalidArgument(input, err), nil
	}

	if params.Directory == "" {
		params.Directory = "."
	}

	log.Debugf("Running %s in %s", params.Command, params.Directory)

	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/c", params.Command)
	} else {
		cmd = exec.Command("/bin/sh", "-c", params.Command)
	}

	var (
		out bytes.Buffer
		pw  = progressWriter{
			out: progress,
		}
		combined = io.MultiWriter(&out, &pw)
	)

	if envvars, err := getWorkspaceEnvFileContents(env); err == nil {
		env = append(env, envvars...)
	}

	cmd.Env = env
	cmd.Dir = params.Directory
	cmd.Stdout = combined
	cmd.Stderr = combined
	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("ERROR: %s\nOUTPUT:\n%s", err, &out), nil
	}
	return out.String(), nil
}

type progressWriter struct {
	out chan<- string
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	pw.out <- string(p)
	return len(p), nil
}

func getWorkspaceEnvFileContents(envs []string) ([]string, error) {
	dir, err := getWorkspaceDir(envs)
	if err != nil {
		return nil, err
	}

	file := filepath.Join(dir, "gptscript.env")

	// Lock the file to prevent concurrent writes from other tool calls.
	locker.RLock(file)
	defer locker.RUnlock(file)

	// This is optional, so no errors are returned if the file does not exist.
	log.Debugf("Reading file %s", file)
	data, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		log.Debugf("The file %s does not exist", file)
		return []string{}, nil
	} else if err != nil {
		log.Debugf("Failed to read file %s: %v", file, err.Error())
		return []string{}, nil
	}

	lines := strings.Split(string(data), "\n")
	var envContents []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "=") {
			envContents = append(envContents, line)
		}
	}

	return envContents, nil

}

func getWorkspaceDir(envs []string) (string, error) {
	for _, env := range envs {
		dir, ok := strings.CutPrefix(env, "GPTSCRIPT_WORKSPACE_DIR=")
		if ok && dir != "" {
			return dir, nil
		}
	}
	return "", fmt.Errorf("no workspace directory found in env")
}

func SysLs(_ context.Context, _ []string, input string, _ chan<- string) (string, error) {
	var params struct {
		Dir string `json:"dir,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return invalidArgument(input, err), nil
	}

	dir := params.Dir
	if dir == "" {
		dir = "."
	}

	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Sprintf("directory does not exist: %s", params.Dir), nil
	} else if err != nil {
		return fmt.Sprintf("Failed to read directory %s: %v", params.Dir, err.Error()), nil
	}

	var result []string
	for _, entry := range entries {
		if entry.IsDir() {
			result = append(result, entry.Name()+"/")
		} else {
			result = append(result, entry.Name())
		}
	}

	if len(result) == 0 {
		return "Empty directory", nil
	}

	return strings.Join(result, "\n"), nil
}

func SysRead(_ context.Context, _ []string, input string, _ chan<- string) (string, error) {
	var params struct {
		Filename string `json:"filename,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return invalidArgument(input, err), nil
	}

	file := params.Filename

	// Lock the file to prevent concurrent writes from other tool calls.
	locker.RLock(file)
	defer locker.RUnlock(file)

	log.Debugf("Reading file %s", file)
	data, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Sprintf("The file %s does not exist", params.Filename), nil
	} else if err != nil {
		return fmt.Sprintf("Failed to read file %s: %v", params.Filename, err.Error()), nil
	}

	if len(data) == 0 {
		return fmt.Sprintf("The file %s has no contents", params.Filename), nil
	}
	return string(data), nil
}

func SysWrite(_ context.Context, _ []string, input string, _ chan<- string) (string, error) {
	var params struct {
		Filename string `json:"filename,omitempty"`
		Content  string `json:"content,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return invalidArgument(input, err), nil
	}

	file := params.Filename

	// Lock the file to prevent concurrent writes from other tool calls.
	locker.Lock(file)
	defer locker.Unlock(file)

	dir := filepath.Dir(file)
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		log.Debugf("Creating dir %s", dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Sprintf("Failed to create directory %s: %v", dir, err.Error()), nil
		}
	}

	data := []byte(params.Content)
	log.Debugf("Wrote %d bytes to file %s", len(data), file)

	if err := os.WriteFile(file, data, 0644); err != nil {
		return fmt.Sprintf("Failed to write file %s: %v", file, err.Error()), nil
	}
	return fmt.Sprintf("Wrote (%d) bytes to file %s", len(data), file), nil
}

func SysAppend(_ context.Context, _ []string, input string, _ chan<- string) (string, error) {
	var params struct {
		Filename string `json:"filename,omitempty"`
		Content  string `json:"content,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return invalidArgument(input, err), nil
	}

	// Lock the file to prevent concurrent writes from other tool calls.
	locker.Lock(params.Filename)
	defer locker.Unlock(params.Filename)

	f, err := os.OpenFile(params.Filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Sprintf("Failed to open file %s: %v", params.Filename, err.Error()), nil
	}

	// Attempt to append to the file and close it immediately.
	// Write is guaranteed to return an error when n != len([]byte(params.Content))
	n, err := f.Write([]byte(params.Content))
	if err := errors.Join(err, f.Close()); err != nil {
		return fmt.Sprintf("Failed to write file %s: %v", params.Filename, err.Error()), nil
	}

	log.Debugf("Appended %d bytes to file %s", n, params.Filename)
	return fmt.Sprintf("Appended (%d) bytes to file %s", n, params.Filename), nil
}

func urlExt(u string) string {
	url, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return filepath.Ext(url.Path)
}

func fixQueries(u string) string {
	url, err := url.Parse(u)
	if err != nil {
		return u
	}
	url.RawQuery = url.Query().Encode()
	return url.String()
}

func SysHTTPGet(_ context.Context, _ []string, input string, _ chan<- string) (_ string, err error) {
	var params struct {
		URL string `json:"url,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return invalidArgument(input, err), nil
	}

	params.URL = fixQueries(params.URL)

	c := http.Client{Timeout: 10 * time.Second}

	log.Debugf("http get %s", params.URL)
	resp, err := c.Get(params.URL)
	if err != nil {
		return fmt.Sprintf("Failed to fetch URL %s: %v", params.URL, err), nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("failed to download %s: %s", params.URL, resp.Status), nil
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Failed to download URL %s: %v", params.URL, err), nil
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return fmt.Sprintf("URL %s has no contents", params.URL), nil
	}

	return string(data), nil
}

func SysHTTPHtml2Text(ctx context.Context, env []string, input string, progress chan<- string) (string, error) {
	content, err := SysHTTPGet(ctx, env, input, progress)
	if err != nil {
		return "", err
	}
	return html2text.FromString(content, html2text.Options{
		PrettyTables: true,
	})
}

func SysHTTPPost(ctx context.Context, _ []string, input string, _ chan<- string) (_ string, err error) {
	var params struct {
		URL         string `json:"url,omitempty"`
		Content     string `json:"content,omitempty"`
		ContentType string `json:"contentType,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return invalidArgument(input, err), nil
	}

	params.URL = fixQueries(params.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, params.URL, strings.NewReader(params.Content))
	if err != nil {
		return "", err
	}
	if params.ContentType != "" {
		req.Header.Set("Content-Type", params.ContentType)
	}

	c := http.Client{Timeout: 10 * time.Second}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Sprintf("Failed to post URL %s: %v", params.URL, err), nil
	}
	defer resp.Body.Close()

	_, _ = io.ReadAll(resp.Body)
	if resp.StatusCode > 399 {
		return fmt.Sprintf("Failed to post URL %s: %s", params.URL, resp.Status), nil
	}

	return fmt.Sprintf("Wrote %d to %s", len([]byte(params.Content)), params.URL), nil
}

func DiscardProgress() (progress chan<- string, closeFunc func()) {
	ch := make(chan string)
	go func() {
		for range ch {
		}
	}()
	return ch, func() {
		close(ch)
	}
}

func SysGetenv(_ context.Context, env []string, input string, _ chan<- string) (string, error) {
	var params struct {
		Name string `json:"name,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return invalidArgument(input, err), nil
	}

	log.Debugf("looking up env var %s", params.Name)
	for _, env := range env {
		k, v, ok := strings.Cut(env, "=")
		if ok && k == params.Name {
			log.Debugf("found env var %s in local environment", params.Name)
			return v, nil
		}
	}

	value := os.Getenv(params.Name)
	if value == "" {
		return fmt.Sprintf("%s is not set or has no value", params.Name), nil
	}
	return value, nil
}

func invalidArgument(input string, err error) string {
	return fmt.Sprintf("Failed to parse arguments %s: %v", input, err)
}

func SysContext(ctx context.Context, _ []string, _ string, _ chan<- string) (string, error) {
	engineContext, _ := engine.FromContext(ctx)

	callContext := *engineContext.GetCallContext()
	callContext.ID = ""
	callContext.ParentID = ""
	data, err := json.Marshal(map[string]any{
		"program": engineContext.Program,
		"call":    callContext,
	})
	if err != nil {
		return invalidArgument("", err), nil
	}

	return string(data), nil
}

func SysChatHistory(ctx context.Context, _ []string, _ string, _ chan<- string) (string, error) {
	engineContext, _ := engine.FromContext(ctx)

	data, err := json.Marshal(engine.ChatHistory{
		History: writeHistory(engineContext),
	})
	if err != nil {
		return invalidArgument("", err), nil
	}

	return string(data), nil
}

func writeHistory(ctx *engine.Context) (result []engine.ChatHistoryCall) {
	if ctx == nil {
		return nil
	}
	if ctx.Parent != nil {
		result = append(result, writeHistory(ctx.Parent)...)
	}
	if ctx.LastReturn != nil && ctx.LastReturn.State != nil {
		result = append(result, engine.ChatHistoryCall{
			ID:         ctx.ID,
			Tool:       ctx.Tool,
			Completion: ctx.LastReturn.State.Completion,
		})
	}
	return
}

func SysChatCurrent(ctx context.Context, _ []string, _ string, _ chan<- string) (string, error) {
	engineContext, _ := engine.FromContext(ctx)

	var call any
	if engineContext != nil && engineContext.CurrentReturn != nil && engineContext.CurrentReturn.State != nil {
		call = engine.ChatHistoryCall{
			ID:         engineContext.ID,
			Tool:       engineContext.Tool,
			Completion: engineContext.CurrentReturn.State.Completion,
		}
	} else {
		call = map[string]any{}
	}

	data, err := json.Marshal(call)
	if err != nil {
		return invalidArgument("", err), nil
	}

	return string(data), nil
}

func SysChatFinish(_ context.Context, _ []string, input string, _ chan<- string) (string, error) {
	var params struct {
		Message string `json:"return,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", &engine.ErrChatFinish{
			Message: input,
		}
	}
	return "", &engine.ErrChatFinish{
		Message: params.Message,
	}
}

func SysAbort(_ context.Context, _ []string, input string, _ chan<- string) (string, error) {
	var params struct {
		Message string `json:"message,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("ABORT: %s", input)
	}
	return "", fmt.Errorf("ABORT: %s", params.Message)
}

func SysRemove(_ context.Context, _ []string, input string, _ chan<- string) (string, error) {
	var params struct {
		Location string `json:"location,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return invalidArgument(input, err), nil
	}

	// Lock the file to prevent concurrent writes from other tool calls.
	locker.Lock(params.Location)
	defer locker.Unlock(params.Location)

	if err := os.Remove(params.Location); err != nil {
		return fmt.Sprintf("Failed to removed %s: %v", params.Location, err), nil
	}

	return fmt.Sprintf("Removed file: %s", params.Location), nil
}

func SysStat(_ context.Context, _ []string, input string, _ chan<- string) (string, error) {
	var params struct {
		Filepath string `json:"filepath,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return invalidArgument(input, err), nil
	}

	stat, err := os.Stat(params.Filepath)
	if err != nil {
		return fmt.Sprintf("failed to stat %s: %s", params.Filepath, err), nil
	}

	title := "File"
	if stat.IsDir() {
		title = "Directory"
	}
	return fmt.Sprintf("%s %s mode: %s, size: %d bytes, modtime: %s", title, params.Filepath, stat.Mode().String(), stat.Size(), stat.ModTime().String()), nil
}

func SysDownload(_ context.Context, env []string, input string, _ chan<- string) (_ string, err error) {
	var params struct {
		URL      string `json:"url,omitempty"`
		Location string `json:"location,omitempty"`
		Override string `json:"override,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return invalidArgument(input, err), nil
	}

	params.URL = fixQueries(params.URL)

	checkExists := true
	tmpDir, err := getWorkspaceDir(env)
	if err != nil {
		return "", err
	}

	if params.Location != "" {
		if s, err := os.Stat(params.Location); err == nil && s.IsDir() {
			tmpDir = params.Location
			params.Location = ""
		}
	}

	if params.Location == "" {
		f, err := os.CreateTemp(tmpDir, "gpt-download*"+urlExt(params.URL))
		if err != nil {
			return fmt.Sprintf("Failed to create temporary file: %s", err), nil
		}
		if err := f.Close(); err != nil {
			return fmt.Sprintf("Failed to close temporary file %s: %v", f.Name(), err), nil
		}
		checkExists = false
		params.Location = f.Name()
	}

	if checkExists && params.Override != "true" {
		if _, err := os.Stat(params.Location); err == nil {
			return fmt.Sprintf("file %s already exists and can not be overwritten", params.Location), nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Sprintf("failed to stat file %s: %v", params.Location, err), nil
		}
	}

	log.Infof("download [%s] to [%s]", params.URL, params.Location)
	resp, err := http.Get(params.URL)
	if err != nil {
		return fmt.Sprintf("failed to download %s: %v", params.URL, err), nil
	}
	defer func() {
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode > 299 {
		return fmt.Sprintf("invalid status code [%d] downloading [%s]: %s", resp.StatusCode, params.URL, resp.Status), nil
	}

	_ = os.Remove(params.Location)
	f, err := os.Create(params.Location)
	if err != nil {
		return fmt.Sprintf("failed to create [%s]: %v", params.Location, err), nil
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Sprintf("failed copying data from [%s] to [%s]: %v", params.URL, params.Location, err), nil
	}

	return fmt.Sprintf("Downloaded %s to %s", params.URL, params.Location), nil
}

func SysTimeNow(context.Context, []string, string, chan<- string) (string, error) {
	return time.Now().Format(time.RFC3339), nil
}
