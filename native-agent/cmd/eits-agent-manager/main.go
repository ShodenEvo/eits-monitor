//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/eitsio/eits-agent/internal/windowsapp"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var version = "0.5.0-alpha.1"

func main() {
	var mw *walk.MainWindow
	var serviceState, serverURL, deviceName, registration, lastReport *walk.Label
	refresh := func() {
		s := windowsapp.QueryStatus()
		serviceState.SetText(s.ServiceState)
		serverURL.SetText(s.ServerURL)
		deviceName.SetText(s.DeviceName)
		if s.Registered {
			registration.SetText("Registered")
		} else {
			registration.SetText("Not registered")
		}
		lastReport.SetText(s.LastLogLine)
	}
	action := func(name string) {
		if err := windowsapp.RunServiceAction(name); err != nil {
			walk.MsgBox(mw, "EITS Agent", err.Error(), walk.MsgBoxIconError)
		}
		refresh()
	}
	openPath := func(path string) { _ = exec.Command("explorer.exe", path).Start() }

	err := (MainWindow{
		AssignTo: &mw,
		Title:    "EITS Agent Manager " + version,
		MinSize:  Size{Width: 620, Height: 430},
		Layout:   VBox{MarginsZero: false},
		Children: []Widget{
			GroupBox{Title: "Agent status", Layout: Grid{Columns: 2}, Children: []Widget{
				Label{Text: "Windows service:"}, Label{AssignTo: &serviceState},
				Label{Text: "Registration:"}, Label{AssignTo: &registration},
				Label{Text: "Server:"}, Label{AssignTo: &serverURL},
				Label{Text: "Device:"}, Label{AssignTo: &deviceName},
			}},
			GroupBox{Title: "Latest activity", Layout: VBox{}, Children: []Widget{
				Label{AssignTo: &lastReport, Text: "No log data yet"},
			}},
			Composite{Layout: Flow{}, Children: []Widget{
				PushButton{Text: "Refresh", OnClicked: refresh},
				PushButton{Text: "Start", OnClicked: func() { action("start") }},
				PushButton{Text: "Stop", OnClicked: func() { action("stop") }},
				PushButton{Text: "Restart", OnClicked: func() { action("restart") }},
				PushButton{Text: "Check for updates", OnClicked: func() {
					updater := filepath.Join(windowsapp.InstallDir(), "eits-agent-updater.exe")
					if err := exec.Command(updater, "--interactive").Start(); err != nil {
						walk.MsgBox(mw, "Update", err.Error(), walk.MsgBoxIconError)
					}
				}},
			}},
			Composite{Layout: Flow{}, Children: []Widget{
				PushButton{Text: "Open logs", OnClicked: func() { openPath(filepath.Join(windowsapp.ProgramDataDir(), "logs")) }},
				PushButton{Text: "Open configuration", OnClicked: func() { openPath(windowsapp.ProgramDataDir()) }},
				PushButton{Text: "Open dashboard", OnClicked: func() {
					s := windowsapp.QueryStatus()
					if s.ServerURL != "" {
						_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", s.ServerURL).Start()
					}
				}},
				PushButton{Text: "About", OnClicked: func() {
					walk.MsgBox(mw, "EITS Agent Manager", fmt.Sprintf("Version %s\nService-based monitoring with automatic updates.", version), walk.MsgBoxIconInformation)
				}},
			}},
		},
	}).Create()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	refresh()
	mw.Run()
}
