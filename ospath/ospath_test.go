package ospath

import (
	"github.com/stretchr/testify/require"
	"log"
	"testing"
)

func TestWindowsPath(t *testing.T) {
	paths := make(map[string]string)
	paths["/foo/moo.txt"] = `\foo\moo.txt`
	paths["/c/foo.ext"] = `C:\foo.ext`
	paths["/C/foo/bar/baz.ext"] = `C:\foo\bar\baz.ext`
	paths[`C:\foo\bar\baz.ext`] = `C:\foo\bar\baz.ext`
	paths[`a:config.sys`] = `A:config.sys`
	paths[`s:foo\moo\goo.ext`] = `S:foo\moo\goo.ext`
	paths[`\\localhost\c$\tmp\foo`] = `C:\tmp\foo`
	paths["//./D/fleem/"] = `D:\fleem\`
	paths["//./D$/fleem/"] = `D:\fleem\`
	paths["//./D:/fleem/"] = `D:\fleem\`

	for src, win := range paths {
		ret := WindowsPath(src)
		log.Printf("%s -> %s\n", src, ret)
		require.Equal(t, win, ret)
	}
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
	w = WindowsPath(`\\localhost\c$\tmp\foo`)
	require.Equal(t, `C:\tmp\foo`, w)
	w = WindowsPath("//./D/fleem/")
	require.Equal(t, `D:\fleem\`, w)
	w = WindowsPath("//./D$/fleem/")
	require.Equal(t, `D:\fleem\`, w)
	w = WindowsPath("//./D:/fleem/")
	require.Equal(t, `D:\fleem\`, w)
}

func TestUnixPath(t *testing.T) {

	paths := make(map[string]string)
	paths["/foo/moo.txt"] = "/foo/moo.txt"
	paths["/c/foo.ext"] = "/c/foo.ext"
	paths["/C/foo/bar/baz.ext"] = "/C/foo/bar/baz.ext"
	paths[`C:\foo\bar\baz.ext`] = "/c/foo/bar/baz.ext"
	paths[`a:config.sys`] = "/a/config.sys"
	paths[`s:foo\moo\goo.ext`] = "/s/foo/moo/goo.ext"
	paths[`\\localhost\c$\tmp\foo`] = "/c/tmp/foo"
	paths["//./D/fleem/"] = "/d/fleem/"
	paths["//./D$/fleem/"] = "/d/fleem/"
	paths["//./D:/fleem/"] = "/d/fleem/"

	for src, unix := range paths {
		ret := UnixPath(src)
		log.Printf("%s -> %s\n", src, ret)
		require.Equal(t, unix, ret)
	}
}
