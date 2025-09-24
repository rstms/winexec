package server

import (
	"bytes"
	"log"
	"os/exec"
	"strings"
)

func (s *WinexecServer) runCommand(label, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	var obuf bytes.Buffer
	var ebuf bytes.Buffer
	cmd.Stdout = &obuf
	cmd.Stderr = &ebuf
	if s.verbose {
		log.Printf("running %s: %v\n", label, command)
	}
	err := cmd.Run()
	if s.verbose {
		ostr := strings.TrimSpace(obuf.String())
		if ostr != "" {
			log.Printf("%s stdout: %s\n", label, ostr)
		}
		estr := strings.TrimSpace(ebuf.String())
		if estr != "" {
			log.Printf("%s stderr: %s\n", label, estr)
		}
	}
	return err
}
