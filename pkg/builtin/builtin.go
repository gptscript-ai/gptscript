package builtin

import (
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
	"sort"
	"strings"

	"github.com/BurntSushi/locker"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/jaytaylor/html2text"
)

var tools = map[string]types.Tool{
	"sys.read": {
		Parameters: types.Parameters{
			Description: "Reads the contents of a file",
			Arguments: types.ObjectSchema(
				"filename", "The name of the file to read"),
		},
		BuiltinFunc: SysRead,
	},
	"sys.write": {
		Parameters: types.Parameters{
			Description: "Write the contents to a file",
			Arguments: types.ObjectSchema(
				"filename", "The name of the file to write to",
				"content", "The content to write"),
		},
		BuiltinFunc: SysWrite,
	},
	"sys.append": {
		Parameters: types.Parameters{
			Description: "Appends the contents to a file",
			Arguments: types.ObjectSchema(
				"filename", "The name of the file to append to",
				"content", "The content to append"),
		},
		BuiltinFunc: SysAppend,
	},
	"sys.http.get": {
		Parameters: types.Parameters{
			Description: "Download the contents of a http or https URL",
			Arguments: types.ObjectSchema(
				"url", "The URL to download"),
		},
		BuiltinFunc: SysHTTPGet,
	},
	"sys.http.html2text": {
		Parameters: types.Parameters{
			Description: "Download the contents of a http or https URL returning the content as rendered text converted from HTML",
			Arguments: types.ObjectSchema(
				"url", "The URL to download"),
		},
		BuiltinFunc: SysHTTPHtml2Text,
	},
	"sys.abort": {
		Parameters: types.Parameters{
			Description: "Aborts execution",
			Arguments: types.ObjectSchema(
				"message", "The description of the error or unexpected result that caused abort to be called",
			),
		},
		BuiltinFunc: SysAbort,
	},
	"sys.http.post": {
		Parameters: types.Parameters{
			Description: "Write contents to a http or https URL using the POST method",
			Arguments: types.ObjectSchema(
				"url", "The URL to POST to",
				"content", "The content to POST",
				"contentType", "The \"content type\" of the content such as application/json or text/plain"),
		},
		BuiltinFunc: SysHTTPPost,
	},
	"sys.find": {
		Parameters: types.Parameters{
			Description: "Traverse a directory looking for files that match a pattern in the style of the unix find command",
			Arguments: types.ObjectSchema(
				"pattern", "The file pattern to look for. The pattern is a traditional unix glob format with * matching any character and ? matching a single character",
				"directory", "The directory to search in. The current directory \".\" will be used as the default if no argument is passed",
			),
		},
		BuiltinFunc: SysFind,
	},
	"sys.exec": {
		Parameters: types.Parameters{
			Description: "Execute a command and get the output of the command",
			Arguments: types.ObjectSchema(
				"command", "The command to run including all applicable arguments",
				"directory", "The directory to use as the current working directory of the command. The current directory \".\" will be used if no argument is passed",
			),
		},
		BuiltinFunc: SysExec,
	},
	"sys.getenv": {
		Parameters: types.Parameters{
			Description: "Gets the value of an OS environment variable",
			Arguments: types.ObjectSchema(
				"name", "The environment variable name to lookup"),
		},
		BuiltinFunc: SysGetenv,
	},
	"sys.download": {
		Parameters: types.Parameters{
			Description: "Downloads a URL, saving the contents to disk at a given location",
			Arguments: types.ObjectSchema(
				"url", "The URL to download, either http or https.",
				"location", "(optional) The on disk location to store the file. If no location is specified a temp location will be used. If the target file already exists it will fail unless override is set to true.",
				"override", "If true and a file at the location exists, the file will be overwritten, otherwise fail. Default is false"),
		},
		BuiltinFunc: SysDownload,
	},
	"sys.remove": {
		Parameters: types.Parameters{
			Description: "Removes the specified files",
			Arguments: types.ObjectSchema(
				"location", "The file to remove"),
		},
		BuiltinFunc: SysRemove,
	},
}

func SysProgram() *types.Program {
	result := &types.Program{
		ToolSet: types.ToolSet{},
	}
	for _, tool := range ListTools() {
		result.ToolSet[tool.ID] = tool
	}
	return result
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
	name, dontFail := strings.CutSuffix(name, "?")
	t, ok := tools[name]
	t.Parameters.Name = name
	t.ID = name
	t.Instructions = "#!" + name
	if ok && dontFail {
		orig := t.BuiltinFunc
		t.BuiltinFunc = func(ctx context.Context, env []string, input string) (string, error) {
			s, err := orig(ctx, env, input)
			if err != nil {
				return fmt.Sprintf("ERROR: %s", err.Error()), nil
			}
			return s, err
		}
	}
	return SetDefaults(t), ok
}

func SysFind(ctx context.Context, env []string, input string) (string, error) {
	var result []string
	var params struct {
		Pattern   string `json:"pattern,omitempty"`
		Directory string `json:"directory,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	if params.Directory == "" {
		params.Directory = "."
	}

	log.Debugf("Finding files %s in %s", params.Pattern, params.Directory)
	err := fs.WalkDir(os.DirFS(params.Directory), ".", func(pathname string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if ok, err := filepath.Match(params.Pattern, d.Name()); err != nil {
			return err
		} else if ok {
			result = append(result, filepath.Join(params.Directory, pathname))
		}
		return nil
	})
	if err != nil {
		return "", nil
	}
	sort.Strings(result)
	return strings.Join(result, "\n"), nil
}

func SysExec(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		Command   string `json:"command,omitempty"`
		Directory string `json:"directory,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	if params.Directory == "" {
		params.Directory = "."
	}

	log.Debugf("Running %s in %s", params.Command, params.Directory)

	cmd := exec.Command("/bin/sh", "-c", params.Command)
	cmd.Env = env
	cmd.Dir = params.Directory
	out, err := cmd.CombinedOutput()
	if err != nil {
		_, _ = os.Stdout.Write(out)
	}
	return string(out), err
}

func SysRead(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		Filename string `json:"filename,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	// Lock the file to prevent concurrent writes from other tool calls.
	locker.RLock(params.Filename)
	defer locker.RUnlock(params.Filename)

	log.Debugf("Reading file %s", params.Filename)
	data, err := os.ReadFile(params.Filename)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func SysWrite(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		Filename string `json:"filename,omitempty"`
		Content  string `json:"content,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	// Lock the file to prevent concurrent writes from other tool calls.
	locker.Lock(params.Filename)
	defer locker.Unlock(params.Filename)

	data := []byte(params.Content)
	log.Debugf("Wrote %d bytes to file %s", len(data), params.Filename)

	return "", os.WriteFile(params.Filename, data, 0644)
}

func SysAppend(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		Filename string `json:"filename,omitempty"`
		Content  string `json:"content,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	// Lock the file to prevent concurrent writes from other tool calls.
	locker.Lock(params.Filename)
	defer locker.Unlock(params.Filename)

	f, err := os.OpenFile(params.Filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return "", err
	}

	// Attempt to append to the file and close it immediately.
	// Write is guaranteed to return an error when n != len([]byte(params.Content))
	n, err := f.Write([]byte(params.Content))
	if err := errors.Join(err, f.Close()); err != nil {
		return "", err
	}

	log.Debugf("Appended %d bytes to file %s", n, params.Filename)

	return "", nil
}

func fixQueries(u string) (string, error) {
	url, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	url.RawQuery = url.Query().Encode()
	return url.String(), nil
}

func SysHTTPGet(ctx context.Context, env []string, input string) (_ string, err error) {
	var params struct {
		URL string `json:"url,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	params.URL, err = fixQueries(params.URL)
	if err != nil {
		return "", err
	}

	log.Debugf("http get %s", params.URL)
	resp, err := http.Get(params.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download %s: %s", params.URL, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func SysHTTPHtml2Text(ctx context.Context, env []string, input string) (string, error) {
	content, err := SysHTTPGet(ctx, env, input)
	if err != nil {
		return "", err
	}
	return html2text.FromString(content, html2text.Options{
		PrettyTables: true,
	})
}

func SysHTTPPost(ctx context.Context, env []string, input string) (_ string, err error) {
	var params struct {
		URL         string `json:"url,omitempty"`
		Content     string `json:"content,omitempty"`
		ContentType string `json:"contentType,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	params.URL, err = fixQueries(params.URL)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, params.URL, strings.NewReader(params.Content))
	if err != nil {
		return "", err
	}
	if params.ContentType != "" {
		req.Header.Set("Content-Type", params.ContentType)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	_, _ = io.ReadAll(resp.Body)
	if resp.StatusCode > 399 {
		return "", fmt.Errorf("failed to post %s: %s", params.URL, resp.Status)
	}

	return fmt.Sprintf("Wrote %d to %s", len([]byte(params.Content)), params.URL), nil
}

func SysGetenv(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		Name string `json:"name,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}
	log.Debugf("looking up env var %s", params.Name)
	return os.Getenv(params.Name), nil
}

func SysAbort(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		Message string `json:"message,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}
	return "", fmt.Errorf("ABORT: %s", params.Message)
}

func SysRemove(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		Location string `json:"location,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	// Lock the file to prevent concurrent writes from other tool calls.
	locker.Lock(params.Location)
	defer locker.Unlock(params.Location)

	return fmt.Sprintf("Removed file: %s", params.Location), os.Remove(params.Location)
}

func SysDownload(ctx context.Context, env []string, input string) (_ string, err error) {
	var params struct {
		URL      string `json:"url,omitempty"`
		Location string `json:"location,omitempty"`
		Override string `json:"override,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	params.URL, err = fixQueries(params.URL)
	if err != nil {
		return "", err
	}

	checkExists := true
	if params.Location == "" {
		f, err := os.CreateTemp("", "gpt-download")
		if err != nil {
			return "", err
		}
		if err := f.Close(); err != nil {
			return "", err
		}
		checkExists = false
		params.Location = f.Name()
	}

	if checkExists && params.Override != "true" {
		if _, err := os.Stat(params.Location); err == nil {
			return "", fmt.Errorf("file %s already exists and can not be overwritten", params.Location)
		} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
	}

	log.Infof("download [%s] to [%s]", params.URL, params.Location)
	resp, err := http.Get(params.URL)
	if err != nil {
		return "", err
	}
	defer func() {
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode > 299 {
		return "", fmt.Errorf("invalid status code [%d] downloading [%s]: %s", resp.StatusCode, params.URL, resp.Status)
	}

	_ = os.Remove(params.Location)
	f, err := os.Create(params.Location)
	if err != nil {
		return "", fmt.Errorf("failed to create [%s]: %w", params.Location, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("failed copying data from [%s] to [%s]: %w", params.URL, params.Location, err)
	}

	return params.Location, nil
}
