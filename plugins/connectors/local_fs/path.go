/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package local_fs

import (
	"path/filepath"
	"sort"
	"strings"

	log "github.com/cihub/seelog"
)

// deduplicatePaths removes duplicate and descendant paths when ancestor paths are present
// For example, if paths contain "/home/user", "/home/user", and "/home/user/documents",
// only "/home/user" will be returned since it covers the duplicate and "/home/user/documents"
func deduplicatePaths(paths []string) []string {
	if len(paths) <= 1 {
		return paths
	}

	// Clean paths and remove duplicates using a map
	pathSet := make(map[string]bool)
	var cleanPaths []string
	for _, path := range paths {
		cleanPath := filepath.Clean(path)
		if cleanPath != "" && !pathSet[cleanPath] {
			pathSet[cleanPath] = true
			cleanPaths = append(cleanPaths, cleanPath)
		}
	}

	if len(cleanPaths) == 0 {
		return []string{}
	}

	// Sort paths to process shorter (ancestor) paths first
	// This helps identify ancestors more easily
	sort.Strings(cleanPaths)

	var result []string
	for i, currentPath := range cleanPaths {
		isDescendant := false

		// Check if current path is a descendant of any already processed path
		for j := 0; j < i; j++ {
			ancestorPath := cleanPaths[j]
			if isPathDescendant(currentPath, ancestorPath) {
				isDescendant = true
				log.Debugf("[%v connector] Skipping descendant path %s (ancestor: %s)", ConnectorLocalFs, currentPath, ancestorPath)
				break
			}
		}

		if !isDescendant {
			result = append(result, currentPath)
		}
	}

	return result
}

// isPathDescendant checks if childPath is a descendant of parentPath
func isPathDescendant(childPath, parentPath string) bool {
	// Clean both paths
	childPath = filepath.Clean(childPath)
	parentPath = filepath.Clean(parentPath)

	// Same path is not considered descendant
	if childPath == parentPath {
		return false
	}

	// Convert both paths to use the same separator for comparison
	childPath = filepath.ToSlash(childPath)
	parentPath = filepath.ToSlash(parentPath)

	// Ensure parent path ends with separator for proper prefix matching
	if !strings.HasSuffix(parentPath, "/") {
		parentPath += "/"
	}

	// Check if child path starts with parent path
	if strings.HasPrefix(childPath, parentPath) {
		return true
	}

	// Also check using filepath.Rel for additional validation
	relPath, err := filepath.Rel(filepath.FromSlash(parentPath[:len(parentPath)-1]), filepath.FromSlash(childPath))
	if err != nil {
		return false
	}

	// If relative path starts with "..", child is not under parent
	if strings.HasPrefix(relPath, "..") {
		return false
	}

	// If relative path is ".", they are the same (already handled above)
	if relPath == "." {
		return false
	}

	// Otherwise, child is a descendant of parent
	return true
}
