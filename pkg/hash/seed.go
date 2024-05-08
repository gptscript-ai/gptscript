package hash

import (
	"encoding/json"
	"hash/fnv"
)

func Seed(input any) int {
	h := fnv.New32a()
	if err := json.NewEncoder(h).Encode(input); err != nil {
		panic(err)
	}
	return int(h.Sum32())
}
