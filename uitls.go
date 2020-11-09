package gotit

import (
	"fmt"
	"os"
	"time"
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

func fmtDuration(d time.Duration) string {
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func fmtProgress(p []int64, size int64) string {
	if len(p) == 0 {
		p = []int64{0, 0}
	}
	cols := 204
	r := make([]rune, 204)
	c := [2]rune{'#', '-'}
	ci := 0
	idx := func(offset int64) int {
		return int(offset * int64(cols) / size)
	}
	for i, _ := range p {
		start := idx(p[i])
		var end int
		if i == len(p)-1 {
			end = idx(size)
		} else {
			end = idx(p[i+1])
		}
		for j := start; j < end; j++ {
			r[j] = c[ci]
		}
		ci = (ci + 1) % 2
	}
	return string(r)
}
