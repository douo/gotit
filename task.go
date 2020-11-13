package gotit

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
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
	metaFile  os.File
	config    Config
	startTime time.Time
}

// Return current status of task
func (t *Task) status() Status {
	d := time.Now().Sub(t.startTime)
	sum := t.meta.sum()
	speed := float64(sum) / d.Seconds()
	return Status{
		filename: t.meta.File,
		speed:    int64(speed),
		size:     t.meta.Size,
		percent:  float32(sum) / float32(t.meta.Size),
		remain:   time.Duration(float64(t.meta.Size-sum)/float64(speed)*1000) * time.Millisecond,
		elapse:   d,
		progress: t.meta.Progress,
	}
}

type Block struct {
	byteValue []byte
	offset    int64
	err       error
}

func (t *Task) Resume() error {
	file, err := os.OpenFile(t.meta.File, os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	mf, err := os.OpenFile(metaFile(t.meta.File), os.O_CREATE|os.O_WRONLY, 0611)
	t.metaFile = *mf
	if err != nil {
		return err
	}
	defer t.metaFile.Close()

	seg := make([]int64, len(t.meta.Progress)/2)
	conn := t.config.MaxConn
	for i := 0; i < len(seg); i++ {
		var start int64
		if i*2+2 >= len(t.meta.Progress) {
			start = t.meta.Size
		} else {
			start = t.meta.Progress[i*2+2]
		}
		seg[i] = start - t.meta.Progress[i*2+1]
	}
	log.Println(seg, conn)
	part := repartition(seg, conn)

	ch := make(chan *Block, 1)
	wg := sync.WaitGroup{}
	for i := 0; i < len(seg); i++ {
		conn = minInt(part[i], int(seg[i]/t.config.MinSplitSize))
		segment := fastRepartition(seg[i], conn)
		log.Println(part[i], segment)
		offset := t.meta.Progress[2*i+1]
		for j := 0; j < len(segment); j++ {
			length := segment[j]
			go doRequest(ch, t.client, t.meta.Url, offset, length, t.config.BufSize)
			offset += segment[j]
			wg.Add(conn)
		}
	}
	t.startTime = time.Now()
	go func() {
		for {
			b := <-ch
			if b != nil {
				file.WriteAt(b.byteValue, b.offset)
				t.meta.updateProgress(b.offset, b.offset+int64(len(b.byteValue)))
				t.status().fmtStatusLine(os.Stdout)
				t.meta.Save(t.metaFile)
			} else {
				wg.Done()
			}
		}
	}()
	wg.Wait()

	return nil

}

// n 个连续的大小不等的段，m 个链接
// 每个链接只下载一个连续段
// 按段的大小分配链接 m
// n = 1  min(m,size_n/split)
// m <= n :[1,1,....0,0] m 个 1, n-m 个 0
// m > n n = [51,50,1,1] m = 6 -> [2,2,1,1]
func repartition(s []int64, c int) []int {
	result := make([]int, len(s))
	if c <= len(s) {
		// m <= n :[1,1,....0,0] m 个 1, n-m 个 0
		for i := 0; i < c; i++ {
			result[i] = 1
		}
	} else {
		c = c + 1 - len(s) // - len 是因为每个 segment 至少有一个链接

		sum := int64(0)
		for i := 0; i < len(s); i++ {
			sum += s[i]
		}
		for i := 0; i < len(s); i++ {
			result[i] = int(s[i]*int64(c)/sum) + 1
		}
		type R struct {
			v int64
			i int
		}
		r := make([]R, len(s))
		for i := 0; i < len(s); i++ {
			r[i] = R{
				v: (s[i] * int64(c)) % sum,
				i: i,
			}
		}
		sort.Slice(r, func(i, j int) bool { return r[i].v > r[j].v })

		sumR := int64(0)
		for i := 0; i < len(s); i++ {
			sumR += r[i].v
		}
		n := int(sumR/sum) - 1

		for i := 0; i < n; i++ {
			result[r[i].i] += 1
		}
	}
	return result
}

// 将 total 平均分为 c 份
func fastRepartition(total int64, c int) (result []int64) {
	r := int(total % int64(c)) //remainder
	s := total / int64(c)
	result = make([]int64, c)
	for i := 0; i < c; i++ {
		if i < r {
			result[i] = s + 1
		} else {
			result[i] = s
		}
	}
	return
}

var TaskExist error = errors.New("task already exist")

func (t *Task) Start() error {
	file, err := allocate(t.meta.File, int64(t.meta.Size))
	if err != nil {
		return err
	}
	defer file.Close()

	mf, err := os.OpenFile(metaFile(t.meta.File), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0611)
	t.metaFile = *mf
	if err != nil {
		return err
	}
	defer t.metaFile.Close()

	var conn int
	if t.meta.SupportRange {
		conn = minInt(t.config.MaxConn, int(t.meta.Size/t.config.MinSplitSize))
	} else {
		conn = 1
	}
	segment := fastRepartition(t.meta.Size, conn)
	log.Print(segment)
	ch := make(chan *Block, 1)
	wg := sync.WaitGroup{}
	var offset int64 = 0
	for i := 0; i < len(segment); i++ {
		length := segment[i]
		go doRequest(ch, t.client, t.meta.Url, offset, length, t.config.BufSize)
		offset += segment[i]
		wg.Add(conn)
	}
	t.startTime = time.Now()
	go func() {
		for {
			b := <-ch
			if b != nil {
				file.WriteAt(b.byteValue, b.offset)
				t.meta.updateProgress(b.offset, b.offset+int64(len(b.byteValue)))
				t.status().fmtStatusLine(os.Stdout)
				t.meta.Save(t.metaFile)
			} else {
				wg.Done()
			}
		}
	}()
	wg.Wait()
	return nil
}

func doRequest(ch chan *Block, client *http.Client, url string, offset int64, length int64, bufSize int) {
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
	ch <- nil
}

func ResumeTask(output string, config Config) (*Task, error) {
	meta := &Meta{}
	err := meta.Restore(output)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	// detech header
	return &Task{
		client: client,
		meta:   meta,
		config: config,
	}, nil
}

func NewTask(url string, output string, config Config) (*Task, error) {
	meta := &Meta{}
	err := meta.Restore(output)
	if err == nil {
		return nil, TaskExist
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
	return err == TaskExist
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
