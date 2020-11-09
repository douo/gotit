package gotit

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/detailyang/go-fallocate"
)

type Config struct {
	MinSplitSize int64
	MaxConn      int
	BufSize      int
}

type Task struct {
	client    *http.Client
	meta      *Meta
	config    Config
	startTime time.Time
}

// View model of task status
type Status struct {
	filename string
	size     int64
	speed    int64
	duration time.Duration
	progress []int64
}

// Return current status of task
func (t *Task) status() Status {
	d := time.Now().Sub(t.startTime)
	return Status{
		filename: t.meta.File,
		speed:    int64(float64(t.meta.sum()) / d.Seconds()),
		size:     t.meta.Size,
		duration: d,
		progress: t.meta.Progress,
	}
}

func (s Status) pretty(w io.Writer) {
	// FYI https://zh.wikipedia.org/wiki/ANSI%E8%BD%AC%E4%B9%89%E5%BA%8F%E5%88%97
	fmt.Print(w, "\x1b[1A\x1b[2K\x1b[F\x1b[2K")
	fmt.Fprintf(w, "%v\t%v/S\n", fmtDuration(s.duration), fmtSize(s.speed))
	fmt.Fprintln(w, fmtProgress(s.progress, s.size))
}

type Block struct {
	byteValue []byte
	offset    int64
}

var errTaskExist error = errors.New("task already exist")

func (t *Task) Start() error {
	file, err := allocate(t.meta.File, int64(t.meta.Size))
	if err != nil {
		return nil
	}
	log.Printf("allocaed:%d", t.meta.Size)

	var conn int
	if t.meta.SupportRange {
		conn = minInt(t.config.MaxConn, int(t.meta.Size/t.config.MinSplitSize))
	} else {
		conn = 1
	}
	remainder := int(t.meta.Size % int64(conn))
	splitSize := t.meta.Size / int64(conn)

	ch := make(chan *Block)
	result := make(chan int)

	for i := 0; i < conn; i++ {
		var offset int64
		var length int64
		if i < remainder {
			offset = int64(i) * (splitSize + 1)
			if i < conn-1 {
				length = splitSize
			} else {
				length = splitSize + 1
			}
			go doRequest(ch, result, t.client, t.meta.Url, offset, length, t.config.BufSize)
		} else {
			offset = int64(i)*splitSize + int64(remainder)
			if i < conn-1 {
				length = splitSize - 1
			} else {
				length = splitSize
			}

			go doRequest(ch, result, t.client, t.meta.Url, offset, length, t.config.BufSize)
		}
	}
	t.startTime = time.Now()
	count := 0
	for {
		select {
		case b := <-ch:
			file.WriteAt(b.byteValue, b.offset)
			t.status().pretty(os.Stdout)
			t.meta.updateProgress(b.offset, b.offset+int64(len(b.byteValue)))

			// t.meta.Save()
		case c := <-result:
			log.Printf("result:%d", c)
			count += 1
			if count >= conn {
				return nil
			}
		}
	}

}

func doRequest(ch chan *Block, result chan int, client *http.Client, url string, offset int64, length int64, bufSize int) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length))
	log.Printf("download:%v", req.Header)
	resp, _ := client.Do(req)
	buf := make([]byte, bufSize)
	var n int
	var err error
	idx := 0
	for n >= 0 && err != io.ErrUnexpectedEOF && err != io.EOF {
		n, err = io.ReadFull(resp.Body, buf)
		if n > 0 {
			cpy := make([]byte, n)
			copy(cpy, buf)
			ch <- &Block{byteValue: cpy, offset: offset}
			offset += int64(n)
			idx = (idx + 1) % 2
		}

	}
	result <- 1
}

func NewTask(url string, output string, config Config) (*Task, error) {
	meta := &Meta{}
	err := meta.Restore(output)
	if err == nil {
		return nil, errTaskExist
	}

	client := &http.Client{}
	// detech header
	head, err := client.Head(url)
	if err != nil {
		return nil, err
	}
	supportHttpRange := supportHttpRange(head)
	length, _ := contentLength(head)

	meta.Url = url
	meta.File = output
	meta.Size = int64(length)
	meta.Etag = head.Header.Get("Etag")
	meta.LastModified = head.Header.Get("Last-Modified")
	meta.SupportRange = supportHttpRange && length > 0
	return &Task{
		client: client,
		meta:   meta,
		config: config,
	}, nil
}

func IsTaskExist(err error) bool {
	return err == errTaskExist
}

func contentLength(resp *http.Response) (i int64, err error) {
	s := resp.Header.Get("Content-Length")
	return strconv.ParseInt(s, 10, 64)
}

func supportHttpRange(resp *http.Response) bool {
	acceptRanges := resp.Header.Get("Accept-Ranges")
	return acceptRanges == "bytes"
}

func allocate(fd string, size int64) (*os.File, error) {
	file, err := os.OpenFile(fd, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	err = fallocate.Fallocate(file, 0, int64(size))
	if err != nil {
		return nil, err
	}
	return file, nil
}
