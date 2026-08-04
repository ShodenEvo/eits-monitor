package queue

import (
	"bufio"
	"encoding/base64"
	"os"
	"path/filepath"
)

type Queue struct {
	Path           string
	MaximumRecords int
}

func (q Queue) Add(payload []byte) error {
	if err := os.MkdirAll(filepath.Dir(q.Path), 0700); err != nil {
		return err
	}
	lines, _ := q.ReadAll()
	lines = append(lines, payload)
	if len(lines) > q.MaximumRecords {
		lines = lines[len(lines)-q.MaximumRecords:]
	}
	return q.Replace(lines)
}

func (q Queue) ReadAll() ([][]byte, error) {
	f, err := os.Open(q.Path)
	if os.IsNotExist(err) {
		return [][]byte{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows := [][]byte{}
	scanner := bufio.NewScanner(f)
	buffer := make([]byte, 64*1024)
	scanner.Buffer(buffer, 4*1024*1024)
	for scanner.Scan() {
		data, err := base64.StdEncoding.DecodeString(scanner.Text())
		if err == nil {
			rows = append(rows, data)
		}
	}
	return rows, scanner.Err()
}

func (q Queue) Replace(rows [][]byte) error {
	if len(rows) == 0 {
		if err := os.Remove(q.Path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(q.Path), 0700); err != nil {
		return err
	}
	tmp := q.Path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for _, row := range rows {
		_, _ = w.WriteString(base64.StdEncoding.EncodeToString(row) + "\n")
	}
	if err := w.Flush(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, q.Path)
}
