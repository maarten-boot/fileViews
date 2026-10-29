//go:build !unix

package listing

import (
	"errors"
	"io/fs"
)

func ownership(fs.FileInfo) (links uint64, owner, group string) {
	return 0, "", ""
}

// FreeSpace is not implemented on this platform.
func FreeSpace(string) (uint64, error) {
	return 0, errors.New("free space: not supported on this platform")
}
