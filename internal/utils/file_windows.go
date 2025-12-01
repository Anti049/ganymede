//go:build windows

package utils

import (
	"fmt"
	"syscall"
	"unsafe"
)

func GetFreeSpaceOfDirectory(path string) (int64, error) {
	h := syscall.MustLoadDLL("kernel32.dll")
	c := h.MustFindProc("GetDiskFreeSpaceExW")

	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes int64

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, fmt.Errorf("error converting path: %v", err)
	}

	r1, _, err := c.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)

	if r1 == 0 {
		return 0, fmt.Errorf("error getting free space of directory: %v", err)
	}

	return freeBytesAvailable, nil
}
