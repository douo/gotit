package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/detailyang/go-fallocate"
)

const MIN_SPLITE_SIZE = 1 * 1024 * 1024 // 1Mb
const MAX_CONN = 10
const BUF_SIZE = 1 * 1024 * 1024

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Print(os.Args)
	err := download(os.Args[1], "/tmp/gotit")
	log.Print(err)
}

type Temp struct {
	file   string
	size   int64
	finish []int64 // 表示已经完成下载的数据库 [start,end) 为一对
}

func (t *Temp) Save() error {
	tf := tempfile(t.file)
	byteValue, err := json.Marshal(t)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(tf, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0611)
	if err != nil {
		return err
	}
	_, err = f.Write(byteValue)
	return err
}

func tempfile(file string) string {
	dir, file := filepath.Split(file)
	return fmt.Sprintf("%s.%s.tmp", dir, file)
}

func (t *Temp) Restore(target string) error {
	f, err := os.Open(tempfile(target))
	if err != nil {
		return err
	}
	byteValue, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	return json.Unmarshal(byteValue, t)
}

func allocate(fd string, size int64) (*os.File, error) {
	file, err := os.OpenFile(fd, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	err = fallocate.Fallocate(file, 0, size)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func makesureTemp(target string) (*Temp, error) {
	tempFile := tempfile(target)
	if isFileExist(tempFile) {
		f, err := os.Open(tempFile)
		if err != nil {
			return nil, err
		}
		bytes, _ := ioutil.ReadAll(f)
		var temp Temp
		json.Unmarshal(bytes, &temp)
		return &temp, nil
	} else {
		return &Temp{file: target}, nil
	}
}

func download(url string, target string) error {
	// prepare
	if isFileExist(target) && isFileExist(tempfile(target)) {
		log.Print("file exist")
		// return resume()
	}

	client := &http.Client{}
	// detech header
	head, err := client.Head(url)
	if err != nil {
		return err
	}
	supportHttpRange := supportHttpRange(head)
	length, _ := contentLength(head)
	if supportHttpRange && length > 0 {
		return initMultiDownload(client, url, target, length)
	} else {
		log.Print("download directly")
	}

	return nil

}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func initMultiDownload(client *http.Client, url string, target string, contentLength int64) error {
	//check temp
	temp := &Temp{file: target, size: contentLength}
	file, err := allocate(target, contentLength)
	if err != nil {
		return err
	}
	log.Printf("allocaed:%d", contentLength)

	expectConnection := Max(MAX_CONN, int((contentLength / MIN_SPLITE_SIZE)))
	remainder := int(contentLength % int64(expectConnection))
	size := contentLength / int64(expectConnection)
	ch := make(chan *Block)
	result := make(chan int)
	for i := 0; i < expectConnection; i++ {
		if i < remainder {
			l := size + 1
			go doDownload(ch, result, client, url, int64(i)*l, l)
		} else {
			l := size
			go doDownload(ch, result, client, url, int64(i)*l+int64(remainder), l)
		}
	}
	count := 0
	for {
		select {
		case b := <-ch:
			time.Sleep(1000)
			file.WriteAt(b.byteValue, b.offset)
			temp.Save()
		case c := <-result:
			log.Printf("result:%d", c)
			count += 1
			if count >= expectConnection {
				return nil
			}
		}
	}

}

type Block struct {
	byteValue []byte
	offset    int64
}

func doDownload(ch chan *Block, result chan int, client *http.Client, url string, offset int64, length int64) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length))
	log.Printf("download:%v", req.Header)
	resp, _ := client.Do(req)
	buf := make([]byte, BUF_SIZE)
	var n int
	var err error
	idx := 0
	log.Printf("d:%d %v %v", n, err, (n >= 0 && err != io.EOF))
	for n >= 0 && err != io.EOF {
		n, err = resp.Body.Read(buf)
		cpy := make([]byte, n)
		copy(cpy, buf)
		ch <- &Block{byteValue: cpy, offset: offset}
		offset += int64(n)
		idx = (idx + 1) % 2
		log.Printf("d:%d %v", n, err)

	}
	result <- 1
}

func contentLength(resp *http.Response) (i int64, err error) {
	s := resp.Header.Get("Content-Length")
	return strconv.ParseInt(s, 10, 64)
}

func supportHttpRange(resp *http.Response) bool {
	acceptRanges := resp.Header.Get("Accept-Ranges")
	return acceptRanges == "bytes"
}
