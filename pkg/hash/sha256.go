package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func Encode(obj any) string {
	data, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	asMap := map[string]any{}
	if err := json.Unmarshal(data, &asMap); err != nil {
		panic(err)
	}

	data, err = json.Marshal(asMap)
	if err != nil {
		panic(err)
	}

	hash := sha256.Sum224(data)
	return hex.EncodeToString(hash[:])
}
