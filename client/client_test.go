package client

import (
	"bytes"
	"github.com/rstms/winexec/message"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
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
	require.NotEmpty(t, os.Getenv("WINEXEC_HOST"))
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

func TestWinexecClientFileDownload(t *testing.T) {
	c := initClient(t)
	testUserFile := ViperGetString("user_file")
	_, filename := filepath.Split(testUserFile)
	dst := filepath.Join("testdata", "files", filename)
	err := c.Download(dst, testUserFile)
	require.Nil(t, err)

	dst = filepath.Join("testdata", "files", "hosts")
	err = c.Download(dst, "/c/windows/system32/drivers/etc/hosts")
	require.Nil(t, err)
}

func TestWinexecClientDirFiles(t *testing.T) {
	c := initClient(t)
	testFilesDir := ViperGetString("files_dir")
	files, err := c.DirFiles(testFilesDir)
	require.Nil(t, err)
	require.IsType(t, []string{}, files)
	require.NotEmpty(t, files)
	for _, file := range files {
		require.NotEmpty(t, file)
		log.Println(file)
	}
}

func TestWinexecClientDirSubs(t *testing.T) {
	c := initClient(t)
	testSubsDir := ViperGetString("subs_dir")
	subs, err := c.DirSubs(testSubsDir)
	require.Nil(t, err)
	require.IsType(t, []string{}, subs)
	require.NotEmpty(t, subs)
	for _, sub := range subs {
		require.NotEmpty(t, sub)
		log.Println(sub)
	}
}

func TestWinexecClientDirEntries(t *testing.T) {
	c := initClient(t)
	entries, err := c.DirEntries(`c:\tmp`)
	require.Nil(t, err)
	require.IsType(t, map[string]message.DirectoryEntry{}, entries)
	require.NotEmpty(t, entries)
	for name, entry := range entries {
		require.IsType(t, message.DirectoryEntry{}, entry)
		require.IsType(t, "", entry.Name)
		require.IsType(t, int64(0), entry.Size)
		require.IsType(t, fs.FileMode(0), entry.Mode)
		require.IsType(t, time.Time{}, entry.ModTime)
		log.Printf("\n%s: %+v\n", name, entry)
		log.Printf("\tName: %s\n", entry.Name)
		log.Printf("\tSize: %d\n", entry.Size)
		log.Printf("\tModTime: %s\n", entry.ModTime.Format(time.DateTime))
		log.Printf("\tIsDir: %v\n", entry.Mode.IsDir())
		log.Printf("\tIsRegular: %v\n", entry.Mode.IsRegular())

	}
}

func TestWinexecClientMkdir(t *testing.T) {
	c := initClient(t)
	err := c.RemoveAll("/c/tmp/foo")
	before, err := c.DirSubs("/c/tmp")
	require.Nil(t, err)
	err = c.MkdirAll("/c/tmp/foo/moo", 0700)
	require.Nil(t, err)
	after, err := c.DirSubs("/c/tmp")
	require.Nil(t, err)
	expected := append(before, "foo")
	slices.Sort(expected)
	require.Equal(t, expected, after)
	subs, err := c.DirSubs("/c/tmp/foo")
	require.Nil(t, err)
	require.Equal(t, []string{"moo"}, subs)
	err = c.RemoveAll("/c/tmp/foo")
	require.Nil(t, err)
}

func TestWinexecClientUploadFile(t *testing.T) {
	c := initClient(t)

	testDir := "/c/tmp/upload_test"
	filename := "config.yaml"
	c.RemoveAll(testDir)
	err := c.MkdirAll(testDir, 0700)
	require.Nil(t, err)
	before, err := c.DirFiles(testDir)
	require.Nil(t, err)
	present := slices.Contains(before, filename)
	require.False(t, present)
	localSrc := filepath.Join("testdata", filename)
	remoteDst := filepath.Join(testDir, filename)
	err = c.Upload(remoteDst, localSrc, false)
	require.Nil(t, err)
	after, err := c.DirFiles(testDir)
	require.Nil(t, err)
	present = slices.Contains(after, filename)
	require.True(t, present)
	checkFile := filepath.Join("testdata", "files", filename)
	err = c.Download(checkFile, remoteDst)
	require.Nil(t, err)

	localData, err := os.ReadFile(localSrc)
	require.Nil(t, err)
	readbackData, err := os.ReadFile(checkFile)
	require.Nil(t, err)
	require.Equal(t, localData, readbackData)

	err = c.RemoveAll(testDir)
	require.Nil(t, err)
}

func TestWinexecClientGetISOQuickDelete(t *testing.T) {
	c := initClient(t)
	testURL := ViperGetString("test_iso_url")
	seconds := 5
	err := c.GetISO("/c/tmp/testfile.iso", testURL, "", "", "", &seconds)
	require.Nil(t, err)
}

func TestWinexecClientGetISODefaultDelete(t *testing.T) {
	c := initClient(t)
	testURL := ViperGetString("test_iso_url")
	err := c.GetISO("/c/tmp/testfile_default.iso", testURL, "", "", "", nil)
	require.Nil(t, err)
}
