package types

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/shlex"
)

func GetToolRefInput(prg *Program, ref ToolReference, input string) (string, error) {
	if ref.Arg == "" {
		return "", nil
	}

	targetArgs := prg.ToolSet[ref.ToolID].Arguments
	targetKeys := map[string]string{}

	if ref.Arg == "*" {
		return input, nil
	}

	if targetArgs == nil {
		return "", nil
	}

	for targetKey := range targetArgs.Properties {
		targetKeys[strings.ToLower(targetKey)] = targetKey
	}

	inputMap := map[string]interface{}{}
	outputMap := map[string]interface{}{}

	_ = json.Unmarshal([]byte(input), &inputMap)
	for k, v := range inputMap {
		inputMap[strings.ToLower(k)] = v
	}

	fields, err := shlex.Split(ref.Arg)
	if err != nil {
		return "", fmt.Errorf("invalid tool args %q: %v", ref.Arg, err)
	}

	for i := 0; i < len(fields); i++ {
		field := fields[i]
		if field == "and" {
			continue
		}
		if field == "as" {
			i++
			continue
		}

		var (
			keyName string
			val     any
		)

		if strings.HasPrefix(field, "$") {
			key := strings.TrimPrefix(field, "$")
			key = strings.TrimPrefix(key, "{")
			key = strings.TrimSuffix(key, "}")
			val = inputMap[strings.ToLower(key)]
		} else {
			val = field
		}

		if len(fields) > i+1 && fields[i+1] == "as" {
			keyName = strings.ToLower(fields[i+2])
		}

		if len(targetKeys) == 0 {
			return "", fmt.Errorf("can not assign arg to context because target tool [%s] has no defined args", ref.ToolID)
		}

		if keyName == "" {
			if len(targetKeys) != 1 {
				return "", fmt.Errorf("can not assign arg to context because target tool [%s] does not have one args. You must use \"as\" syntax to map the arg to a key %v", ref.ToolID, targetKeys)
			}
			for k := range targetKeys {
				keyName = k
			}
		}

		if targetKey, ok := targetKeys[strings.ToLower(keyName)]; ok {
			outputMap[targetKey] = val
		} else {
			return "", fmt.Errorf("can not assign arg to context because target tool [%s] does not args [%s]", ref.ToolID, keyName)
		}
	}

	if len(outputMap) == 0 {
		return "", nil
	}

	output, err := json.Marshal(outputMap)
	return string(output), err
}
