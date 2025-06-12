package credentials

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/docker/cli/cli/config/types"
	"github.com/docker/docker-credential-helpers/client"
	"github.com/docker/docker-credential-helpers/credentials"
)

type mockProgram struct {
	// mode is either "db" or "normal"
	// db mode will honor contexts, normal mode will not
	mode     string
	action   string
	contexts []string
}

func (m *mockProgram) Input(in io.Reader) {
	switch m.action {
	case credentials.ActionList:
		var contexts []string
		if err := json.NewDecoder(in).Decode(&contexts); err == nil && len(contexts) > 0 {
			m.contexts = contexts
		}
	}
	// TODO: add other cases here as needed
}

func (m *mockProgram) Output() ([]byte, error) {
	switch m.action {
	case credentials.ActionList:
		switch m.mode {
		case "db":
			// Return only credentials that are in the list of contexts.
			creds := make(map[string]string)
			for _, context := range m.contexts {
				creds[fmt.Sprintf("https://example///%s", context)] = "username"
			}
			return json.Marshal(creds)
		case "normal":
			// Return credentials in the list of contexts, plus some made up extras.
			creds := make(map[string]string)
			for _, context := range m.contexts {
				creds[fmt.Sprintf("https://example///%s", context)] = "username"
			}
			creds[fmt.Sprintf("https://example///%s", "otherContext1")] = "username"
			creds[fmt.Sprintf("https://example///%s", "otherContext2")] = "username"
			return json.Marshal(creds)
		}
	}
	return nil, nil
}

func NewMockProgram(t *testing.T, mode string) client.ProgramFunc {
	return func(args ...string) client.Program {
		p := &mockProgram{
			mode: mode,
		}
		if len(args) > 0 {
			p.action = args[0]
		}
		return p
	}
}

func TestGetAll(t *testing.T) {
	dbProgram := NewMockProgram(t, "db")
	normalProgram := NewMockProgram(t, "normal")

	tests := []struct {
		name     string
		program  client.ProgramFunc
		wantErr  bool
		contexts []string
		expected map[string]types.AuthConfig
	}{
		{name: "db", program: dbProgram, wantErr: false, contexts: []string{"credctx"}, expected: map[string]types.AuthConfig{
			"https://example///credctx": {
				Username:      "username",
				ServerAddress: "https://example///credctx",
			},
		}},
		{name: "normal", program: normalProgram, wantErr: false, contexts: []string{"credctx"}, expected: map[string]types.AuthConfig{
			"https://example///credctx": {
				Username:      "username",
				ServerAddress: "https://example///credctx",
			},
			"https://example///otherContext1": {
				Username:      "username",
				ServerAddress: "https://example///otherContext1",
			},
			"https://example///otherContext2": {
				Username:      "username",
				ServerAddress: "https://example///otherContext2",
			},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &toolCredentialStore{
				program:  test.program,
				contexts: test.contexts,
			}
			got, err := store.GetAll()
			if (err != nil) != test.wantErr {
				t.Errorf("GetAll() error = %v, wantErr %v", err, test.wantErr)
			}
			if len(got) != len(test.expected) {
				t.Errorf("GetAll() got %d credentials, want %d", len(got), len(test.expected))
			}
			for name, cred := range got {
				if _, ok := test.expected[name]; !ok {
					t.Errorf("GetAll() got unexpected credential: %s", name)
				}
				if got[name].Username != test.expected[name].Username {
					t.Errorf("GetAll() got unexpected username for %s", cred.ServerAddress)
				}
				if got[name].Username != test.expected[name].Username {
					t.Errorf("GetAll() got unexpected username for %s", name)
				}
			}
		})
	}
}
