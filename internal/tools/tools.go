package tools

import (
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"
)

func IsGoP() bool {
	return IsExist("go.mod")
}

func IsRustP() bool {
	return IsExist("Cargo.toml")
}

func InArray(in string, arr []string) bool {
	for _, a := range arr {
		if strings.EqualFold(in, a) {
			return true
		}
	}
	return false
}

// File info cache for better performance
var (
	fileInfoCache      = make(map[string]fileInfoCacheEntry)
	fileInfoCacheMutex sync.RWMutex
)

type fileInfoCacheEntry struct {
	modTime  int64
	cachedAt time.Time
}

// GetFileModTime returns unix timestamp of `os.File.ModTime` for the given path with caching
func GetFileModTime(path string) int64 {
	path = strings.Replace(path, "\\", "/", -1)

	// Check cache first (valid for 1 second)
	fileInfoCacheMutex.RLock()
	if entry, exists := fileInfoCache[path]; exists {
		if time.Since(entry.cachedAt) < time.Second {
			fileInfoCacheMutex.RUnlock()
			return entry.modTime
		}
	}
	fileInfoCacheMutex.RUnlock()

	// Get fresh file info
	fi, err := os.Stat(path)
	if err != nil {
		return time.Now().Unix()
	}

	modTime := fi.ModTime().Unix()

	// Update cache
	fileInfoCacheMutex.Lock()
	fileInfoCache[path] = fileInfoCacheEntry{
		modTime:  modTime,
		cachedAt: time.Now(),
	}
	fileInfoCacheMutex.Unlock()

	return modTime
}

func GetPathDir(path string, contain []string) []string {
	var dirs []string
	files, err := os.ReadDir(path)
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
		files, _ := os.ReadDir(p)
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
	return os.WriteFile(file, []byte(content), os.ModePerm)
}

func ReadFile(file string) (string, error) {
	b, err := os.ReadFile(file)
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

// RunCommand executes a shell command and returns the output
func RunCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// GetFileInfo returns file information for the given path
func GetFileInfo(path string) (os.FileInfo, error) {
	return os.Stat(path)
}
