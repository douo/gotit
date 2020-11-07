package gotit

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

type Config struct {
	MinSplitSize uint64
	MaxConn      uint
	BufSize      uint
}

type Task struct {
	client *http.Client
	output string
	meta   *Meta
}

func (t *Task) start() {

}

var errTaskExist error = errors.New("task exist")

func NewTask(url url.URL, output string, config Config) (*Task, error) {
	meta := &Meta{}
	err := meta.Restore(output)
	if err == nil {
		return nil, errTaskExist
	}

	client := &http.Client{}
	// detech header
	head, err := client.Head(url.String())
	if err != nil {
		return nil, err
	}
	supportHttpRange := supportHttpRange(head)
	length, _ := contentLength(head)

	if supportHttpRange && length > 0 {
		meta.url = url.String()
		meta.file = output
		meta.size = uint64(length)
		meta.etag = head.Header.Get("Etag")
		meta.lastModified = head.Header.Get("Last-Modified")
		return &Task{
			client: client,
			output: output,
			meta:   meta,
		}, nil
	}
	return &Task{
		client: client,
		output: output,
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
