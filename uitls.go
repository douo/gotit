package gotit

import "os"

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
