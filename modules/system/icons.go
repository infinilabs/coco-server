/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	"fmt"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (h *APIHandler) getIcons(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	baseDir := global.Env().SystemConfig.WebAppConfig.UI.LocalPath
	if !filepath.IsAbs(baseDir) {
		// Get the path of the executable
		exePath, err := GetExecutablePath()
		if err != nil {
			panic(err)
		}

		// Get the directory of the executable
		exeDir := filepath.Dir(exePath)
		baseDir = filepath.Join(exeDir, baseDir)
	}
	iconsPathPrefix := filepath.Join("/assets", "icons")
	iconsDir := filepath.Join(baseDir, iconsPathPrefix)

	icons, err := readIcons(iconsDir, iconsPathPrefix)
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

func readIcons(iconsDir string, pathPrefix string) ([]IconInfo, error) {
	var icons []IconInfo

	err := filepath.WalkDir(iconsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %v", path, err)
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Skip hidden files
		fileName := d.Name()
		if strings.HasPrefix(fileName, ".") {
			return nil
		}

		// Validate image extensions
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".svg" && ext != ".gif" {
			return nil
		}

		// Extract category (parent directory name)
		category := filepath.Base(filepath.Dir(path))

		// Get relative path
		relPath, err := filepath.Rel(iconsDir, path)
		if err != nil {
			return fmt.Errorf("error getting relative path for %s: %v", path, err)
		}
		icon := IconInfo{
			Category: category,
			Name:     strings.TrimSuffix(fileName, ext),
			Path:     filepath.Join(pathPrefix, relPath), // Ensure consistent path format
		}
		icons = append(icons, icon)
		return nil
	})

	if err != nil {
		return nil, err
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
