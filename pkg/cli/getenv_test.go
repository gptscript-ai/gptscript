package cli

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	// Cleaning up
	defer func(currentEnvValue string) {
		os.Setenv("testKey", currentEnvValue)
	}(os.Getenv("testKey"))

	// Tests
	testCases := []struct {
		name           string
		key            string
		def            string
		envValue       string
		expectedResult string
	}{
		{
			name:           "NoValueUseDefault",
			key:            "testKey",
			def:            "defaultValue",
			envValue:       "",
			expectedResult: "defaultValue",
		},
		{
			name:           "ValueExistsNoCompress",
			key:            "testKey",
			def:            "defaultValue",
			envValue:       "testValue",
			expectedResult: "testValue",
		},
		{
			name:     "ValueExistsCompressed",
			key:      "testKey",
			def:      "defaultValue",
			envValue: `{"_gz":"H4sIAEosrGYC/ytJLS5RKEvMKU0FACtB3ewKAAAA"}`,

			expectedResult: "test value",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			os.Setenv(test.key, test.envValue)

			result := getEnv(test.key, test.def)

			if result != test.expectedResult {
				t.Errorf("expected: %s, got: %s", test.expectedResult, result)
			}
		})
	}
}
