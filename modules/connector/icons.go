/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connector

import (
	httprouter "infini.sh/framework/core/api/router"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (h *APIHandler) getIcons(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	// Get the path of the executable
	exePath, err := GetExecutablePath()
	if err != nil {
		panic(err)
	}

	// Get the directory of the executable
	exeDir := filepath.Dir(exePath)
	icons, err := readIcons(exeDir)
	if err != nil {
		panic(err)
	}

	// Write the list of icon files
	h.WriteJSON(w, icons, http.StatusOK)
}

type IconInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Category string `json:"category"`
}

func readIcons(path string) ([]IconInfo, error) {
	// Get the icons directory
	iconsDir := filepath.Join(path, ".public/assets/connector")

	// Get the list of files in the icons directory
	entries, err := os.ReadDir(iconsDir)
	if err != nil {
		return nil, err
	}

	var icons = make([]IconInfo, 0)
	for _, entry := range entries {
		path := filepath.Join(iconsDir, entry.Name())
		if entry.IsDir() {
			// Get the list of files in the category directory
			categoryEntries, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, categoryEntry := range categoryEntries {
				//skip directories
				if categoryEntry.IsDir() {
					continue
				}
				iconName := categoryEntry.Name()
				//skip hidden files
				if strings.HasPrefix(iconName, ".") {
					continue
				}
				ext := filepath.Ext(iconName)
				icon := IconInfo{
					Category: entry.Name(),
					Name:     strings.TrimSuffix(iconName, ext),
					Path:     filepath.Join("/assets/connector", entry.Name(), iconName),
				}
				icons = append(icons, icon)
			}
		}
	}
	return icons, nil
}

func GetExecutablePath() (string, error) {
	// Get the absolute path of the executable
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Get the directory of the executable
	exeDir := filepath.Dir(exePath)
	fi, err := os.Lstat(exePath)
	if err != nil {
		return "", err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		// Get the absolute path
		realPath, err := filepath.EvalSymlinks(exePath)
		if err != nil {
			return "", err
		}
		return filepath.Dir(realPath), nil
	}
	return exeDir, nil
}
