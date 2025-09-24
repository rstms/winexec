/*
Copyright © 2025 Matt Krueger <mkrueger@rstms.net>
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

 1. Redistributions of source code must retain the above copyright notice,
    this list of conditions and the following disclaimer.

 2. Redistributions in binary form must reproduce the above copyright notice,
    this list of conditions and the following disclaimer in the documentation
    and/or other materials provided with the distribution.

 3. Neither the name of the copyright holder nor the names of its contributors
    may be used to endorse or promote products derived from this software
    without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
POSSIBILITY OF SUCH DAMAGE.
*/
package cmd

import (
	"bytes"
	"fmt"
	"github.com/rstms/winexec/server"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
	"strings"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "run the winexec server",
	Long: `
Implement an HTTPS API server for remote execution and filesystem functions, 
secured by mutual TLS authentication with a private x509 CA
`,
	Run: func(cmd *cobra.Command, args []string) {
		daemon, err := server.NewWinexecServer()
		cobra.CheckErr(err)
		if ViperGetBool("debug") {
			fmt.Println(FormatJSON(daemon.GetConfig()))
		}
		runCommand("startup")
		defer runCommand("shutdown")
		err = daemon.Run()
		cobra.CheckErr(err)
	},
}

func runCommand(state string) {
	key := "server." + state + "_command"
	binary := ViperGetString(key)
	if binary == "" {
		return
	}
	args := ViperGetStringSlice(key + "_args")
	command := exec.Command(binary, args...)
	var obuf bytes.Buffer
	var ebuf bytes.Buffer
	command.Stdout = &obuf
	command.Stderr = &ebuf
	if ViperGetBool("verbose") {
		log.Printf("running %s: %v\n", key, command)
	}
	err := command.Run()
	if ViperGetBool("verbose") {
		ostr := strings.TrimSpace(obuf.String())
		if ostr != "" {
			fmt.Fprintf(os.Stdout, "%s stdout: %s\n", key, ostr)
		}
		estr := strings.TrimSpace(ebuf.String())
		if estr != "" {
			fmt.Fprintf(os.Stderr, "%s stderr: %s\n", key, estr)
		}
	}
	cobra.CheckErr(err)

}

func init() {
	CobraAddCommand(rootCmd, rootCmd, serverCmd)
	OptionString(serverCmd, "bind-address", "a", "127.0.0.1", "bind address")
	OptionString(serverCmd, "https-port", "p", "10080", "listen port")
	OptionString(serverCmd, "ca", "", "", "certificate authority PEM file")
	OptionString(serverCmd, "cert", "", "", "server certificate PEM file")
	OptionString(serverCmd, "key", "", "", "server certificate key PEM file")
}
