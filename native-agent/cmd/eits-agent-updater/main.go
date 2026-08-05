//go:build windows

package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/eitsio/eits-agent/internal/windowsapp"
	"github.com/lxn/walk"
)

var version = "0.5.0-alpha.1"

type Manifest struct {
	Version   string `json:"version"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Mandatory bool   `json:"mandatory"`
	Notes     string `json:"release_notes"`
}

func download(url, path string) error {
	c := http.Client{Timeout: 2 * time.Minute}
	r, err := c.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return fmt.Errorf("download returned %s", r.Status)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r.Body)
	return err
}
func checksum(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), e
}
func unzipFile(src, dst string) error {
	r, e := zip.OpenReader(src)
	if e != nil {
		return e
	}
	defer r.Close()
	for _, f := range r.File {
		p := filepath.Join(dst, f.Name)
		if !strings.HasPrefix(filepath.Clean(p), filepath.Clean(dst)+string(os.PathSeparator)) {
			return errors.New("unsafe archive path")
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(p, 0755)
			continue
		}
		_ = os.MkdirAll(filepath.Dir(p), 0755)
		in, e := f.Open()
		if e != nil {
			return e
		}
		out, e := os.Create(p)
		if e != nil {
			in.Close()
			return e
		}
		_, e = io.Copy(out, in)
		in.Close()
		out.Close()
		if e != nil {
			return e
		}
	}
	return nil
}

func main() {
	interactive := flag.Bool("interactive", false, "show dialogs")
	manifestURL := flag.String("manifest", "https://eits.myds.me/downloads/agent/windows/update.json", "update manifest URL")
	flag.Parse()
	notify := func(title, msg string, err bool) {
		if *interactive {
			icon := walk.MsgBoxIconInformation
			if err {
				icon = walk.MsgBoxIconError
			}
			walk.MsgBox(nil, title, msg, icon)
		} else {
			fmt.Println(msg)
		}
	}
	r, e := http.Get(*manifestURL)
	if e != nil {
		notify("EITS Update", "Unable to check for updates: "+e.Error(), true)
		return
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		notify("EITS Update", "Update server returned "+r.Status, true)
		return
	}
	var m Manifest
	if e = json.NewDecoder(r.Body).Decode(&m); e != nil {
		notify("EITS Update", e.Error(), true)
		return
	}
	if m.Version == "" || m.URL == "" {
		notify("EITS Update", "Invalid update manifest", true)
		return
	}
	if m.Version == version {
		notify("EITS Update", "EITS Agent is already current ("+version+").", false)
		return
	}
	tmp, e := os.MkdirTemp("", "eits-update-")
	if e != nil {
		notify("EITS Update", e.Error(), true)
		return
	}
	defer os.RemoveAll(tmp)
	pkg := filepath.Join(tmp, "update.zip")
	if e = download(m.URL, pkg); e != nil {
		notify("EITS Update", e.Error(), true)
		return
	}
	if m.SHA256 != "" {
		sum, _ := checksum(pkg)
		if !strings.EqualFold(sum, m.SHA256) {
			notify("EITS Update", "Checksum verification failed", true)
			return
		}
	}
	stage := filepath.Join(tmp, "stage")
	_ = os.MkdirAll(stage, 0755)
	if e = unzipFile(pkg, stage); e != nil {
		notify("EITS Update", e.Error(), true)
		return
	}
	install := windowsapp.InstallDir()
	backup := filepath.Join(tmp, "backup")
	_ = os.MkdirAll(backup, 0755)
	_ = windowsapp.RunServiceAction("stop")
	names := []string{"eits-agent-service.exe", "eits-agent-manager.exe", "eits-agent-updater.exe"}
	for _, n := range names {
		old := filepath.Join(install, n)
		if _, e = os.Stat(old); e == nil {
			_ = copyFile(old, filepath.Join(backup, n))
		}
		src := filepath.Join(stage, n)
		if _, e = os.Stat(src); e == nil {
			if e = copyFile(src, old); e != nil {
				rollback(backup, install, names)
				notify("EITS Update", "Update failed and was rolled back: "+e.Error(), true)
				return
			}
		}
	}
	if e = windowsapp.RunServiceAction("start"); e != nil {
		rollback(backup, install, names)
		_ = windowsapp.RunServiceAction("start")
		notify("EITS Update", "Service failed after update; previous version restored.", true)
		return
	}
	notify("EITS Update", "Updated successfully to "+m.Version+".\n\n"+m.Notes, false)
	_ = exec.Command(filepath.Join(install, "eits-agent-manager.exe")).Start()
}
func copyFile(src, dst string) error {
	in, e := os.Open(src)
	if e != nil {
		return e
	}
	defer in.Close()
	out, e := os.Create(dst)
	if e != nil {
		return e
	}
	_, e = io.Copy(out, in)
	if ce := out.Close(); e == nil {
		e = ce
	}
	return e
}
func rollback(src, dst string, names []string) {
	for _, n := range names {
		p := filepath.Join(src, n)
		if _, e := os.Stat(p); e == nil {
			_ = copyFile(p, filepath.Join(dst, n))
		}
	}
}
