//go:build windows
// +build windows

package main

import (
	"syscall"

	"golang.org/x/sys/windows"
)

func getFreeSpace(path string) (uint64, error) {
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	lpDirectoryName, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	err = windows.GetDiskFreeSpaceEx(lpDirectoryName, &freeBytesAvailable, &totalNumberOfBytes, &totalNumberOfFreeBytes)
	if err != nil {
		return 0, err
	}

	return freeBytesAvailable, nil
}
