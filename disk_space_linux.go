//go:build linux
// +build linux

package main

import (
	"syscall"
)

func getFreeSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}

	// Calculate free space available.
	return stat.Bavail * uint64(stat.Bsize), nil
}
