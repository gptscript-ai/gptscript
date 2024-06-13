package internal

import (
	"io/fs"
	"os"
)

var FS fs.FS = defaultFS{}

type defaultFS struct{}

func (d defaultFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}
