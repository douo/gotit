package main

import "net/http"

type SimpleTask struct {
	client *http.Client
	req    *http.Request
	target string
}

type MultiTask struct {
	SimpleTask
	meta *Meta
}

func NewMultiTask(client *http.Client, req *http.Request, inspectHeader *http.Header, target string) (MultiTask, error) {
	meta := &Meta{
		url : req.URL.String(),
		file : target
		size : length
	}

	err := meta.Restore(target)
	if(err ==nil){
		log.Printf("Resume Download")
	}

	task := MultiTask{
		client: client,
		req: req,
		target : target
		meta : &Meta
	}
}

type Task interface {
	Init
}
