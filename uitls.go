package gotit

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func isFileExist(fn string) bool {
	_, err := os.Stat(fn)
	return err == nil || os.IsExist(err)
}

func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func maxUint(x, y uint) uint {
	if x > y {
		return x
	}
	return y
}

func maxUint64(x, y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}

func minInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func minUint(x, y uint) uint {
	if x < y {
		return x
	}
	return y
}

func minUint64(x, y uint64) uint64 {
	if x < y {
		return x
	}
	return y
}

func insertInt64(array []int64, i int, element ...int64) []int64 {
	for i > len(array) {
		array = append(array, 0)
	}
	return append(array[:i], append(element, array[i:]...)...)
}

func fmtSize(s int64) string {
	if s < 1024 {
		return fmt.Sprintf("%dB", s)
	}
	if s < 1024*1024 {
		return fmt.Sprintf("%.2fKb", float64(s)/1024)
	}
	if s < 1024*1024*1024 {
		return fmt.Sprintf("%.2fMb", float64(s)/1024/1024)
	}
	return fmt.Sprintf("%.2fGb", float64(s)/1024/1024/1024)
}

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func getWidth() uint {
	ws := &winsize{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		panic(errno)
	}
	return uint(ws.Col)
}
