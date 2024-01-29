package hash

import "hash/fnv"

func Seed(input any) int {
	s := Encode(input)
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return int(h.Sum32())
}
