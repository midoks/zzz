package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/midoks/zzz/internal/logger"
)

// BuildCache manages build cache to avoid unnecessary rebuilds
type BuildCache struct {
	cacheDir   string
	lastHashes map[string]string
	mutex      sync.RWMutex
}

// NewBuildCache creates a new build cache instance
func NewBuildCache(projectRoot string) *BuildCache {
	cacheDir := filepath.Join(projectRoot, ".zzz-cache")
	os.MkdirAll(cacheDir, 0755)

	return &BuildCache{
		cacheDir:   cacheDir,
		lastHashes: make(map[string]string),
	}
}

// calculateProjectHash calculates a hash of all relevant source files
func (bc *BuildCache) calculateProjectHash(projectRoot string, extensions []string) (string, error) {
	hasher := sha256.New()

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip cache directory and other ignored directories
		if info.IsDir() {
			name := info.Name()
			if name == ".zzz-cache" || name == ".git" || name == "target" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only hash files with relevant extensions
		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		for _, validExt := range extensions {
			if ext == validExt {
				// Add file path and modification time to hash
				relPath, _ := filepath.Rel(projectRoot, path)
				hasher.Write([]byte(relPath))
				hasher.Write([]byte(info.ModTime().Format(time.RFC3339Nano)))
				break
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ShouldRebuild checks if a rebuild is necessary based on file changes
func (bc *BuildCache) ShouldRebuild(projectRoot, language string, extensions []string) bool {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	currentHash, err := bc.calculateProjectHash(projectRoot, extensions)
	if err != nil {
		logger.Log.Warnf("Failed to calculate project hash: %s", err)
		return true // Rebuild on error
	}

	lastHash, exists := bc.lastHashes[language]
	if !exists || lastHash != currentHash {
		bc.lastHashes[language] = currentHash
		logger.Log.Infof("Source files changed, rebuild required")
		return true
	}

	logger.Log.Info("No source changes detected, skipping rebuild")
	return false
}

// MarkBuildComplete marks a successful build completion
func (bc *BuildCache) MarkBuildComplete(language string) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	// Cache is already updated in ShouldRebuild
	logger.Log.Infof("Build cache updated for %s", language)
}

// ClearCache clears the build cache
func (bc *BuildCache) ClearCache() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	bc.lastHashes = make(map[string]string)
	os.RemoveAll(bc.cacheDir)
	os.MkdirAll(bc.cacheDir, 0755)

	logger.Log.Info("Build cache cleared")
}

// GetCacheStats returns cache statistics
func (bc *BuildCache) GetCacheStats() map[string]interface{} {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	return map[string]interface{}{
		"cached_languages": len(bc.lastHashes),
		"cache_directory":  bc.cacheDir,
	}
}
