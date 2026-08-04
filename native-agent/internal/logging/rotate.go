package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type RotatingWriter struct {
	Path     string
	MaxBytes int64
	MaxFiles int
}

func (w *RotatingWriter) Write(p []byte) (int, error) {
	if err := os.MkdirAll(filepath.Dir(w.Path), 0750); err != nil {
		return 0, err
	}
	if info, err := os.Stat(w.Path); err == nil && info.Size()+int64(len(p)) > w.MaxBytes {
		_ = w.rotate()
	}
	f, err := os.OpenFile(w.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return f.Write(p)
}
func (w *RotatingWriter) rotate() error {
	for i := w.MaxFiles - 1; i >= 1; i-- {
		old := fmt.Sprintf("%s.%d", w.Path, i)
		next := fmt.Sprintf("%s.%d", w.Path, i+1)
		_ = os.Rename(old, next)
	}
	_ = os.Remove(fmt.Sprintf("%s.%d", w.Path, w.MaxFiles))
	return os.Rename(w.Path, w.Path+".1")
}
func Multi(path string, maxMB int64, maxFiles int) io.Writer {
	return io.MultiWriter(os.Stdout, &RotatingWriter{Path: path, MaxBytes: maxMB * 1024 * 1024, MaxFiles: maxFiles})
}
