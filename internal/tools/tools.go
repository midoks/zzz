package tools

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func InArray(in string, arr []string) bool {
	for _, a := range arr {
		if strings.EqualFold(in, a) {
			return true
		}
	}
	return false
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
