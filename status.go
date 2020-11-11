package gotit

import (
	"fmt"
	"io"
	"path/filepath"
	"time"
)

// View model of task status
type Status struct {
	filename string
	size     int64
	speed    int64
	percent  float32
	remain   time.Duration
	elapse   time.Duration
	progress []int64
}

//a.txt [####-----##----##----| time ] speed
func (s Status) fmtStatusLine(w io.Writer) {
	fmt.Fprint(w, "\x1b[1A", "\x1b[2K")
	prefix := fmt.Sprintf("%s %.2f%% %s    [",
		filepath.Base(s.filename), s.percent*100, fmtDuration(s.elapse))
	suffix := fmt.Sprintf("| %s ] %v/s\n",
		fmtDuration(s.remain), fmtSize(s.speed))
	cols := int(getWidth())
	fmt.Fprint(w, prefix,
		fmtProgress(s.progress, s.size, cols-len(prefix)-len(suffix)),
		suffix)
}

func fmtDuration(d time.Duration) string {
	if d > time.Hour {
		h := d / time.Hour
		d -= h * time.Hour
		m := d / time.Minute
		d -= m * time.Minute
		s := d / time.Second
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	} else {
		m := d / time.Minute
		d -= m * time.Minute
		s := d / time.Second
		return fmt.Sprintf("%02d:%02d", m, s)
	}
}

func fmtProgress(p []int64, size int64, cols int) string {
	if len(p) == 0 {
		p = []int64{0, 0}
	}
	r := make([]rune, cols)
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
		var j int
		for j = start; j < end; j++ {
			r[j] = c[ci]
		}
		ci = (ci + 1) % 2
	}
	return string(r)
}
