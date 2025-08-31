//go:build !openbsd

package server

import (
	_ "embed"
	"fmt"
	"github.com/getlantern/systray"
	"log"
	"runtime"
)

//go:embed icon.ico
var iconData []byte

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
	mQuit := systray.AddMenuItem(fmt.Sprintf("Quit %v", m.Title), "Shutdown and exit")

	// Handle menu item clicks
	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()

}

func (m *Menu) onExit() {
	log.Printf("onExit: received gui shutdown event\n")
	m.shutdownRequest <- struct{}{}
	log.Printf("onExit: shutdown requested\n")
}
