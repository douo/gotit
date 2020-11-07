package gotit

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Meta struct {
	url          string // session
	file         string
	size         uint64
	etag         string
	lastModified string
	progress     []uint64
}

/*
  convert target file name to meta file name
*/
func metaFile(target string) string {
	dir, file := filepath.Split(target)
	return fmt.Sprintf("%s.%s.meta", dir, file)

}

/*
  try to retore meta from target file
*/
func (m *Meta) Restore(target string) error {
	f, err := os.Open(metaFile(target))
	if err != nil {
		return err
	}
	byteValue, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	return json.Unmarshal(byteValue, m)
}

func (m *Meta) Save() error {
	tf := metaFile(m.file)
	byteValue, err := json.Marshal(m)
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
