//go:build smoke

package smoke

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	openai "github.com/gptscript-ai/chat-completion-client"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/tests/judge"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/icmd"
)

const defaultModelEnvVar = "GPTSCRIPT_DEFAULT_MODEL"

func TestSmoke(t *testing.T) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	smokeJudge, err := judge.New[[]event](client)
	require.NoError(t, err, "error initializing smoke test judge")

	for _, tc := range getTestcases(t) {
		t.Run(tc.name, func(t *testing.T) {
			cmd := icmd.Command(
				"gptscript",
				"--color=false",
				"--disable-cache",
				"--events-stream-to",
				tc.actualEventsFile,
				"--default-model",
				tc.defaultModel,
				tc.gptFile,
			)

			result := icmd.RunCmd(cmd)
			defer func() {
				t.Helper()
				assert.NoError(t, os.Remove(tc.actualEventsFile))
			}()

			require.NoError(t, result.Error, "stderr: %q", result.Stderr())
			require.Zero(t, result.ExitCode)

			var (
				actualEvents   = getActualEvents(t, tc.actualEventsFile)
				expectedEvents = make([]event, 0)
			)
			f, err := os.Open(tc.expectedEventsFile)
			if os.IsNotExist(err) {
				// No expected events found, store the results of the latest call as the golden file for future tests runs
				f, err := os.Create(tc.expectedEventsFile)
				require.NoError(t, err)
				defer f.Close()

				encoder := json.NewEncoder(f)
				encoder.SetIndent("", "    ")
				require.NoError(t, encoder.Encode(actualEvents))
				t.Skipf("Generated initial golden file %q, skipping test", tc.expectedEventsFile)
			} else {
				require.NoError(t, err)
				defer f.Close()

				decoder := json.NewDecoder(f)
				require.NoError(t, decoder.Decode(&expectedEvents))
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			equal, reasoning, err := smokeJudge.Equal(
				ctx,
				expectedEvents,
				actualEvents,
				`
- disregard differences in event order, timestamps, generated IDs, and natural language verbiage, grammar, and punctuation
- compare events with matching event types
- the overall stream of events and set of tools called should roughly match
- arguments passed in tool calls should be roughly the same
- the final callFinish event should be semantically similar
`,
			)
			require.NoError(t, err, "error getting judge ruling on output")
			require.True(t, equal, reasoning)
			t.Logf("reasoning: %q", reasoning)
		})
	}
}

type testcase struct {
	name               string
	dir                string
	gptFile            string
	defaultModel       string
	modelName          string
	env                []string
	actualEventsFile   string
	expectedEventsFile string
}

func getTestcases(t *testing.T) []testcase {
	t.Helper()

	defaultModel := os.Getenv(defaultModelEnvVar)
	modelName := strings.Split(defaultModel, " ")[0]

	var testcases []testcase
	for _, d := range lo.Must(os.ReadDir("testdata")) {
		if !d.IsDir() {
			continue
		}
		var (
			dirName = d.Name()
			dir     = filepath.Join("testdata", dirName)
		)

		files, err := os.ReadDir(dir)
		require.NoError(t, err, "failed to get testdata dir %q", dir)

		for _, f := range files {
			if f.IsDir() || filepath.Ext(f.Name()) != ".gpt" {
				continue
			}

			testcases = append(testcases, testcase{
				name:               dirName,
				dir:                dir,
				gptFile:            filepath.Join(dir, f.Name()),
				defaultModel:       defaultModel,
				modelName:          modelName,
				expectedEventsFile: filepath.Join(dir, fmt.Sprintf("%s-expected.json", modelName)),
				actualEventsFile:   filepath.Join(dir, fmt.Sprintf("%s.json", modelName)),
			})

			// Only take the first .gpt file in each testcase directory
			break
		}
	}

	return testcases
}

type event struct {
	runner.Event
	ChatRequest  *openai.ChatCompletionRequest `json:"chatRequest,omitempty"`
	ChatResponse *types.CompletionMessage      `json:"chatResponse,omitempty"`
}

func getActualEvents(t *testing.T, eventsFile string) []event {
	t.Helper()

	f, err := os.Open(eventsFile)
	require.NoError(t, err)
	defer f.Close()

	var (
		events  []event
		scanner = bufio.NewScanner(f)
	)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip blank lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		var e event
		require.NoError(t, json.Unmarshal([]byte(line), &e))
		events = append(events, e)
	}

	require.NoError(t, scanner.Err())

	return events
}
