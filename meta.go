package gotit

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Meta struct {
	Url          string  `json:"url"` // session
	File         string  `json:"file"`
	Size         int64   `json:"size"`
	Etag         string  `json:"etag"`
	LastModified string  `json:"lastModified"`
	SupportRange bool    `json:"supportRange"`
	Progress     []int64 `json:"progress"`
}

/*
  convert target file name to meta file name
*/
func metaFile(target string) string {
	dir, file := filepath.Split(target)
	return fmt.Sprintf("%s.%s.meta", dir, file)

}

// 1. [s1,e1] [s2, e2] ....
// 2. 由小到大排序
// 3. 每组数之间不相交，出现相交需要合并
func innerUpdateProgress(p []int64, start int64, end int64) []int64 {
	if start == end {
		return p
	}
	var i = 1
	for ; i < len(p); i += 2 {
		if start > p[i] {
			continue
		}
		if start == p[i] {
			// s 等于 ei，扩展 [si,ei] -> [si,end]
			if i+1 < (len(p)-1) && end == p[i+1] {
				// 与下一组数相交
				p[i] = p[i+2]
				copy(p[i:], p[i+2:])
				return p[:len(p)-2]
			} else {
				p[i] = end
			}
			return p
		}
		if start < p[i] {
			//预设 end 小于 si
			if end == p[i-1] {
				// content 刚好重合
				p[i-1] = start
				return p
			}
			// content不会相交
			break
		}
	}
	return insertInt64(p, i-1, start, end)
}

func (m *Meta) updateProgress(start int64, end int64) {
	m.Progress = innerUpdateProgress(m.Progress, start, end)
}

func (m *Meta) sum() int64 {
	p := m.Progress
	sum := int64(0)
	for i := 0; i < len(p); i += 2 {
		sum += p[i+1] - p[i]
	}
	return sum
}

func (m *Meta) pretty() string {
	return ""
}

/*
  try to retore meta from target file
*/
func (m *Meta) Restore(target string) error {
	f, err := os.Open(metaFile(target))
	if err != nil {
		return err
	}
	defer f.Close()
	byteValue, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	return json.Unmarshal(byteValue, m)
}

func (m *Meta) Save() error {
	tf := metaFile(m.File)
	byteValue, err := json.Marshal(m)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(tf, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0611)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(byteValue)
	return err
}
