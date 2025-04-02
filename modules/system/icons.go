/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	_ "embed"
	"fmt"
	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"net/http"
	"os"
	"path"
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
	var (
		icons []IconInfo
		err   error
	)
	if util.FileExists(iconsDir) {
		icons, err = readIcons(iconsDir, iconsPathPrefix)
		if err != nil {
			panic(err)
		}
	}
	// icons from fonts
	iconfontMeta := getIconfontAsset()
	iconsFromFonts := transformIconfont(iconfontMeta)
	for _, icon := range iconsFromFonts {
		icons = append(icons, icon)
	}

	// Write the list of icon files
	h.WriteJSON(w, icons, http.StatusOK)
}

const (
	SourceFonts string = "fonts"
	SourceIcons string = "icons"
)

type IconInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Category string `json:"category"`
	Source   string `json:"source"` // option values: "fonts", "icons"
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
			Source:   SourceIcons,
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

type IconfontAsset struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	FontFamily    string  `json:"font_family"`
	CSSPrefixText string  `json:"css_prefix_text"`
	Description   string  `json:"description"`
	Glyphs        []Glyph `json:"glyphs"`
}

type Glyph struct {
	IconID         string `json:"icon_id"`
	Name           string `json:"name"`
	FontClass      string `json:"font_class"`
	Unicode        string `json:"unicode"`
	UnicodeDecimal int    `json:"unicode_decimal"`
}

func transformIconfont(input []byte) []IconInfo {
	if len(input) == 0 {
		return nil
	}
	var iconfontAsset IconfontAsset
	util.MustFromJSONBytes(input, &iconfontAsset)
	var result []IconInfo
	for _, glyph := range iconfontAsset.Glyphs {
		// Use the font class as the icon name, final icon svg source locate at /assets/fonts/icons/iconfont.js
		result = append(result, IconInfo{
			Source: SourceFonts,
			Name:   glyph.FontClass,
			Path:   glyph.FontClass,
		})
	}
	return result
}

//go:embed iconfont.json
var iconfontFile []byte

func getIconfontAsset() []byte {
	externalConfig := path.Join(global.Env().GetConfigDir(), "iconfont.json")
	if util.FileExists(externalConfig) {
		log.Infof("loading iconfont file from %s", externalConfig)
		bytes, err := util.FileGetContent(externalConfig)
		if err != nil {
			log.Errorf("load iconfont file failed, use embedded config, err: %v", err)
		} else {
			iconfontFile = bytes
		}
	}
	return iconfontFile
}
