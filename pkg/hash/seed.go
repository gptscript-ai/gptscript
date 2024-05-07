package hash

import (
	"encoding/gob"
	"hash/fnv"
)

func Seed(input any) int {
	h := fnv.New32a()
	if err := gob.NewEncoder(h).Encode(input); err != nil {
		panic(err)
	}
	return int(h.Sum32())
}
