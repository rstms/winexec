//go:build !openbsd

package server

import (
	_ "embed"
	"github.com/rstms/systray"
	"log"
	"runtime"
)

type Menu struct {
	Title            string
	shutdownRequest  chan struct{}
	shutdownComplete chan struct{}
}

func NewMenu(title string, shutdown, complete chan struct{}) (*Menu, error) {
	m := Menu{
		Title:            title,
		shutdownRequest:  shutdown,
		shutdownComplete: complete,
	}
	// Ensure the program is run with a Windows GUI context
	runtime.LockOSThread()
	systray.Run(m.onReady, m.onExit)
	return &m, nil
}

func (m *Menu) onReady() {
	// Set the icon and tooltip
	systray.SetTitle(m.Title)
	systray.SetTooltip(m.Title)
	systray.SetIcon(iconData)

	// Add menu items
	systray.AddMenuItem("winexec v"+Version, "winexec server daemon")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Shutdown", "shutdown server and exit")
	mPing := systray.AddMenuItem("Ping", "write log message")

	// Handle menu item clicks
	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			case <-mPing.ClickedCh:
				log.Println("ping")
			}
		}
	}()

}

func (m *Menu) onExit() {
	log.Printf("onExit: received exit event\n")
	m.shutdownRequest <- struct{}{}
	log.Printf("onExit: sent shutdown request\n")
}
