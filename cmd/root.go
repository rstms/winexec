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
	"embed"
	"fmt"
	common "github.com/rstms/go-common"
	"github.com/rstms/winexec/server"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var cfgFile string

//go:embed certs/*
var certs embed.FS

var rootCmd = &cobra.Command{
	Use:     "winexec",
	Version: server.Version,
	Short:   "user session remote command execution daemon",
	Long: `
Run an HTTPS server under the logged-in 'on the glass' user sesssion.
Endpoints provide authorized clients to execute a command line in this
context.  Any GUI programs started interact with the desktop as expected.
An icon is displated in the 'task notification area'.
`,
	Run: func(cmd *cobra.Command, args []string) {
		daemon, err := server.NewDaemon(cmd.Name(), certs)
		cobra.CheckErr(err)
		err = daemon.Start()
		cobra.CheckErr(err)
		sigint := make(chan os.Signal)
		signal.Notify(sigint, syscall.SIGINT)
		if ViperGetBool("verbose") {
			fmt.Println("CTRL-C to shutdown")
		}
		<-sigint
		err = daemon.Stop()
		cobra.CheckErr(err)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	common.Init("winexec", server.Version)
	cobra.OnInitialize(InitConfig)
	OptionString(rootCmd, "config", "c", "", "config file")
	OptionString(rootCmd, "address", "a", "127.0.0.1", "bind address")
	OptionString(rootCmd, "port", "p", "10080", "listen port")
	OptionString(rootCmd, "logfile", "l", "", "log filename")
	OptionSwitch(rootCmd, "debug", "d", "enable debug diagnostics")
	OptionSwitch(rootCmd, "verbose", "v", "enable diagnostic output")
}
