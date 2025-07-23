package tools

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/midoks/zzz/internal/logger"
)

func IsGoP() bool {
	if IsExist("go.mod") {
		return true
	}
	return false
}

func IsRustP() bool {
	if IsExist("Cargo.toml") {
		return true
	}
	return false
}

func InArray(in string, arr []string) bool {
	for _, a := range arr {
		if strings.EqualFold(in, a) {
			return true
		}
	}
	return false
}

// GetFileModTime returns unix timestamp of `os.File.ModTime` for the given path.
func GetFileModTime(path string) int64 {
	path = strings.Replace(path, "\\", "/", -1)
	f, err := os.Open(path)
	if err != nil {
		logger.Log.Errorf("Failed to open file on '%s': %s", path, err)
		return time.Now().Unix()
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		logger.Log.Errorf("Failed to get file stats: %s", err)
		return time.Now().Unix()
	}

	return fi.ModTime().Unix()
}

func GetPathDir(path string, contain []string) []string {
	var dirs []string
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return dirs
	}

	for _, file := range files {
		if file.IsDir() {
			name := file.Name()
			if InArray(name, contain) {
				continue
			}

			npath := path + "/" + name
			ndirs := GetPathDir(npath, contain)
			dirs = append(dirs, npath)

			for _, f := range ndirs {
				dirs = append(dirs, f)
			}
		}
	}
	dirs = append(dirs, path)
	return dirs
}

func GetVailDir(paths []string, contain []string) []string {
	var newDirs []string
	for _, p := range paths {
		files, _ := ioutil.ReadDir(p)
		for _, f := range files {
			fname := f.Name()
			suffix := path.Ext(fname)
			suffix = strings.Trim(suffix, ".")
			if InArray(suffix, contain) && !InArray(p, newDirs) {
				newDirs = append(newDirs, p)
				break
			}
		}
	}
	return newDirs
}

// IsFile returns true if given path exists as a file (i.e. not a directory).
func IsFile(path string) bool {
	f, e := os.Stat(path)
	if e != nil {
		return false
	}
	return !f.IsDir()
}

// IsDir returns true if given path is a directory, and returns false when it's
// a file or does not exist.
func IsDir(dir string) bool {
	f, e := os.Stat(dir)
	if e != nil {
		return false
	}
	return f.IsDir()
}

// IsExist returns true if a file or directory exists.
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func WriteFile(file string, content string) error {
	return ioutil.WriteFile(file, []byte(content), os.ModePerm)
}

func ReadFile(file string) (string, error) {
	f, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	return string(b), err
}

func Md5Byte(buf []byte) string {
	hash := md5.New()
	hash.Write(buf)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func Md5(s string) string {
	return Md5Byte([]byte(s))
}
