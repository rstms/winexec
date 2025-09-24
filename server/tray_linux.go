//go:build linux

package server

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
	return &m, nil
}
