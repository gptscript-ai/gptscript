package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func ID(parts ...string) string {
	d := sha256.New()
	for i, part := range parts {
		if i > 0 {
			d.Write([]byte{0x00})
		}
		d.Write([]byte(part))
	}
	hash := d.Sum(nil)
	return hex.EncodeToString(hash[:])
}

func Digest(obj any) string {
	data, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

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

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
