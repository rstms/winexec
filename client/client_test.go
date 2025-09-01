package client

import (
	"bytes"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func dumpConfig(t *testing.T) {
	filename := viper.ConfigFileUsed()
	log.Printf("configFileUsed: %s\n", filename)
	dir, err := os.Getwd()
	require.Nil(t, err)
	log.Printf("current directory: %s\n", dir)
	var buf bytes.Buffer
	err = viper.WriteConfigTo(&buf)
	require.Nil(t, err)
	log.Println(buf.String())
}

func initTestConfig(t *testing.T) {
	testFile := filepath.Join("testdata", "config.yaml")
	Init("test", Version, testFile)
	ViperSet("debug", true)
}

func initClient(t *testing.T) *WinexecClient {
	initTestConfig(t)
	c, err := NewWinexecClient()
	require.Nil(t, err)
	return c
}

func TestFileDownload(t *testing.T) {
	c := initClient(t)
	dst := filepath.Join("testdata", "hosts")
	err := c.Download(dst, "/c/users/mkrueger/howdy.txt")
	require.Nil(t, err)
}

func TestWindowsPath(t *testing.T) {
	w := WindowsPath("/foo/moo.txt")
	require.Equal(t, `\foo\moo.txt`, w)
	w = WindowsPath("/c/foo.ext")
	require.Equal(t, `C:\foo.ext`, w)
	w = WindowsPath("/C/foo/bar/baz.ext")
	require.Equal(t, `C:\foo\bar\baz.ext`, w)
	w = WindowsPath(`C:\foo\bar\baz.ext`)
	require.Equal(t, `C:\foo\bar\baz.ext`, w)
	w = WindowsPath(`a:config.sys`)
	require.Equal(t, `A:config.sys`, w)
	w = WindowsPath(`s:foo\moo\goo.ext`)
	require.Equal(t, `S:foo\moo\goo.ext`, w)
}
