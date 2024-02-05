package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/acorn-io/gptscript/pkg/types"
)

var Tools = map[string]types.Tool{
	"sys.read": {
		Description: "Reads the contents of a file",
		Arguments: types.ObjectSchema(
			"filename", "The name of the file to read"),
		BuiltinFunc: SysRead,
	},
	"sys.write": {
		Description: "Write the contents to a file",
		Arguments: types.ObjectSchema(
			"filename", "The name of the file to write to",
			"content", "The content to write"),
		BuiltinFunc: SysWrite,
	},
	"sys.http.get": {
		Description: "Download the contents of a http or https URL",
		Arguments: types.ObjectSchema(
			"url", "The URL to download"),
		BuiltinFunc: SysHTTPGet,
	},
	"sys.abort": {
		Description: "Aborts execution",
		Arguments: types.ObjectSchema(
			"message", "The description of the error or unexpected result that caused abort to be called",
		),
		BuiltinFunc: SysAbort,
	},
	"sys.http.post": {
		Description: "Write contents to a http or https URL using the POST method",
		Arguments: types.ObjectSchema(
			"url", "The URL to POST to",
			"content", "The content to POST",
			"contentType", "The \"content type\" of the content such as application/json or text/plain"),
		BuiltinFunc: SysHTTPPost,
	},
	"sys.find": {
		Description: "Traverse a directory looking for files that match a pattern in the style of the unix find command",
		Arguments: types.ObjectSchema(
			"pattern", "The file pattern to look for. The pattern is a traditional unix glob format with * matching any character and ? matching a single character",
			"directory", "The directory to search in. The current directory \".\" will be used as the default if no argument is passed",
		),
		BuiltinFunc: SysFind,
	},
	"sys.exec": {
		Description: "Execute a command and get the output of the command",
		Arguments: types.ObjectSchema(
			"command", "The command to run including all applicable arguments",
			"directory", "The directory to use as the current working directory of the command. The current directory \".\" will be used if no argument is passed",
		),
		BuiltinFunc: SysExec,
	},
}

func ListTools() (result []types.Tool) {
	var keys []string
	for k := range Tools {
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
	t, ok := Tools[name]
	t.Name = name
	t.ID = name
	t.Instructions = "#!" + name
	return t, ok
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
	err := fs.WalkDir(os.DirFS(params.Directory), params.Directory, func(pathname string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if ok, err := filepath.Match(params.Pattern, d.Name()); err != nil {
			return err
		} else if ok {
			result = append(result, filepath.Join(pathname))
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
	return string(out), err
}

func SysRead(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		Filename string `json:"filename,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

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

	data := []byte(params.Content)
	msg := fmt.Sprintf("Wrote %d bytes to file %s", len(data), params.Filename)
	log.Debugf(msg)

	return "", os.WriteFile(params.Filename, data, 0644)
}

func SysHTTPGet(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		URL string `json:"url,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
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

func SysHTTPPost(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		URL         string `json:"url,omitempty"`
		Content     string `json:"content,omitempty"`
		ContentType string `json:"contentType,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
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

func SysAbort(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		Message string `json:"message,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}
	return "", fmt.Errorf("ABORT: %s", params.Message)
}
