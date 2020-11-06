package main

import (
	"log"
	"net/http"
)

type Config struct {
	MinSplitSize uint64
	MaxConn      uint
	BufSize      uint
}

type Task struct {
	client *http.Client
	req    *http.Request
	target string
}

type MultiTask struct {
	*Task
	meta *Meta
}

func NewMultiTask(client *http.Client, req *http.Request, inspectHeader *http.Header, target string, length uint64) (*MultiTask, error) {
	meta := &Meta{
		url:  req.URL.String(),
		file: target,
		size: length,
	}

	err := meta.Restore(target)
	if err == nil {
		log.Printf("Resume Download")
	}

	task := MultiTask{
		&Task{
			client: client,
			req:    req,
			target: target,
		},
		meta,
	}
	log.Print(task)
	return nil, nil
}
