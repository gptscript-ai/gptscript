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
	hash := sha256.New()
	switch v := obj.(type) {
	case []byte:
		hash.Write(v)
	case string:
		hash.Write([]byte(v))
	default:
		if err := json.NewEncoder(hash).Encode(obj); err != nil {
			panic(err)
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}
