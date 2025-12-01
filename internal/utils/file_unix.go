//go:build !windows

package utils

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func GetFreeSpaceOfDirectory(path string) (int64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("error getting free space of directory: %v", err)
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}
