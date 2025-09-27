package ospath

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
)

const Version = "1.2.3"

// convert a local path to a windows path
func WindowsPath(localPath string) string {
	if strings.Contains(localPath, `\`) {
		//log.Println("has windows separators")
		localPath = strings.ReplaceAll(localPath, `\`, "/")
	}
	var drivePrefix string
	winPath := localPath
	switch {
	case regexp.MustCompile(`^/[a-zA-Z]/`).MatchString(winPath):
		//log.Println("has drive letter coded as dir")
		drivePrefix = strings.ToUpper(string(winPath[1])) + ":"
		winPath = winPath[2:]
	case regexp.MustCompile(`^[a-zA-Z]:`).MatchString(winPath):
		//log.Println("has drive letter colon prefix")
		drivePrefix = strings.ToUpper(string(winPath[0])) + ":"
		winPath = winPath[2:]
	case regexp.MustCompile(`^//[^/]+/[^/]+/[^/]+`).MatchString(winPath):
		//log.Printf("has UNC drive prefix: %s\n", winPath)
		elements := regexp.MustCompile(`^//([^/]+)/([a-zA-Z][$:]{0,1})(/.*)$`).FindStringSubmatch(winPath)
		if len(elements) == 4 {
			/*
				for i, element := range elements {
					log.Printf("%d: %s\n", i, element)
				}
			*/
			drivePrefix = strings.ToUpper(string(elements[2][0])) + ":"
			winPath = elements[3]
		}
	}
	winPath = strings.ReplaceAll(winPath, "/", `\`)
	ret := drivePrefix + winPath
	//log.Printf("localPath=%s\n", localPath)
	//log.Printf("drivePrefix=%s\n", drivePrefix)
	//log.Printf("winPath=%s\n", winPath)
	//log.Printf("ret=%s\n", ret)
	return ret
}

func UnixPath(srcPath string) string {
	unixPath := srcPath
	if strings.HasPrefix(unixPath, `\\`) || strings.HasPrefix(unixPath, "//") {
		unixPath = WindowsPath(unixPath)
	}
	if strings.Contains(unixPath, `\`) {
		unixPath = strings.ReplaceAll(unixPath, `\`, "/")
	}
	if regexp.MustCompile(`^[a-zA-Z]:`).MatchString(unixPath) {
		driveLetter := strings.ToLower(string(unixPath[0]))
		if len(unixPath) < 3 {
			unixPath = ""
		} else {
			unixPath = unixPath[2:]
		}
		unixPath = fmt.Sprintf("/%s/%s", driveLetter, strings.TrimLeft(unixPath, "/"))
	}
	return unixPath
}

func LocalPath(srcPath string) string {
	if runtime.GOOS == "windows" {
		return WindowsPath(srcPath)
	}
	return UnixPath(srcPath)
}
