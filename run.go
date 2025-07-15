/*
Copyright © 2024 Matt Krueger <mkrueger@rstms.net>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package main

import (
	"bytes"
	"log"
	"os/exec"
)

func Spawn(command string) (int, error) {
	cmd := exec.Command("cmd", "/c", command)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	var exitCode int
	if Debug {
		log.Printf("Spawn: %v\n", cmd)
	}
	err := cmd.Run()
	if err != nil {
		switch e := err.(type) {
		case *exec.ExitError:
			exitCode = e.ExitCode()
			err = nil
		}
	}
	if Debug {
		log.Printf("exitCode=%d\n", exitCode)
		log.Printf("err=%v\n", err)
	}
	return exitCode, err
}

func Run(command string, args ...string) (int, string, string, error) {
	if Debug {
		log.Printf("Run: %s %v\n", command, args)
	}
	cmd := exec.Command(command, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	exitCode := 0
	err := cmd.Run()
	if err != nil {
		switch e := err.(type) {
		case *exec.ExitError:
			exitCode = e.ExitCode()
			err = nil
		default:
			return -1, "", "", err
		}
	}
	if Debug {
		log.Printf("exitCode=%d\n", exitCode)
		log.Printf("stdout=%s\n", stdout.String())
		log.Printf("stderr=%s\n", stderr.String())
		log.Printf("err=%v\n", err)
	}
	return exitCode, stdout.String(), stderr.String(), err
}
