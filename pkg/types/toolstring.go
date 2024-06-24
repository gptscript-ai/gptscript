package types

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

func ToDisplayText(tool Tool, input string) string {
	interpreter := tool.GetInterpreter()
	if interpreter == "" {
		return ""
	}

	if strings.HasPrefix(interpreter, "sys.") {
		data := map[string]string{}
		_ = json.Unmarshal([]byte(input), &data)
		out, err := ToSysDisplayString(interpreter, data)
		if err != nil {
			return fmt.Sprintf("Running %s", interpreter)
		}
		return out
	}

	if tool.Source.Repo != nil {
		repo := tool.Source.Repo
		root := strings.TrimPrefix(repo.Root, "https://")
		root = strings.TrimSuffix(root, ".git")
		name := repo.Name
		if name == "tool.gpt" {
			name = ""
		}

		return fmt.Sprintf("Running %s from %s", tool.Name, filepath.Join(root, repo.Path, name))
	}

	if tool.Source.Location != "" {
		return fmt.Sprintf("Running %s from %s", tool.Name, tool.Source.Location)
	}

	return ""
}

func ToSysDisplayString(id string, args map[string]string) (string, error) {
	switch id {
	case "sys.append":
		return fmt.Sprintf("Appending to file `%s`", args["filename"]), nil
	case "sys.download":
		if location := args["location"]; location != "" {
			return fmt.Sprintf("Downloading `%s` to `%s`", args["url"], location), nil
		} else {
			return fmt.Sprintf("Downloading `%s` to workspace", args["url"]), nil
		}
	case "sys.exec":
		return fmt.Sprintf("Running `%s`", args["command"]), nil
	case "sys.find":
		dir := args["directory"]
		if dir == "" {
			dir = "."
		}
		return fmt.Sprintf("Finding `%s` in `%s`", args["pattern"], dir), nil
	case "sys.http.get":
		return fmt.Sprintf("Downloading `%s`", args["url"]), nil
	case "sys.http.post":
		return fmt.Sprintf("Sending to `%s`", args["url"]), nil
	case "sys.http.html2text":
		return fmt.Sprintf("Downloading `%s`", args["url"]), nil
	case "sys.ls":
		return fmt.Sprintf("Listing `%s`", args["dir"]), nil
	case "sys.read":
		return fmt.Sprintf("Reading `%s`", args["filename"]), nil
	case "sys.remove":
		return fmt.Sprintf("Removing `%s`", args["location"]), nil
	case "sys.write":
		return fmt.Sprintf("Writing `%s`", args["filename"]), nil
	case "sys.context", "sys.stat", "sys.getenv", "sys.abort", "sys.chat.current", "sys.chat.finish", "sys.chat.history", "sys.echo", "sys.prompt", "sys.time.now":
		return "", nil
	default:
		return "", fmt.Errorf("unknown tool for display string: %s", id)
	}
}
