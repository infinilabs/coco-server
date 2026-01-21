/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"strings"

	"github.com/EdlinOrg/prominentcolor"
	log "github.com/cihub/seelog"
	"github.com/lucasb-eyer/go-colorful"
	_ "golang.org/x/image/webp"
)

// PredefinedColor represents a color with its English name
type PredefinedColor struct {
	Name  string
	Color colorful.Color
}

// predefinedColors defines the color vocabulary for mapping
var predefinedColors = []PredefinedColor{
	{"red", colorful.Color{R: 1, G: 0, B: 0}},
	{"orange", colorful.Color{R: 1, G: 0.64, B: 0}},
	{"yellow", colorful.Color{R: 1, G: 1, B: 0}},
	{"green", colorful.Color{R: 0, G: 1, B: 0}},
	{"cyan", colorful.Color{R: 0, G: 1, B: 1}},
	{"blue", colorful.Color{R: 0, G: 0, B: 1}},
	{"purple", colorful.Color{R: 0.5, G: 0, B: 0.5}},
	{"pink", colorful.Color{R: 1, G: 0.75, B: 0.79}},
	{"white", colorful.Color{R: 1, G: 1, B: 1}},
	{"gray", colorful.Color{R: 0.5, G: 0.5, B: 0.5}},
	{"black", colorful.Color{R: 0, G: 0, B: 0}},
	{"brown", colorful.Color{R: 0.64, G: 0.16, B: 0.16}},
}

// ExtractDominantColors extracts the top 3 dominant colors from an image
// and returns them as lowercase English color names.
func ExtractDominantColors(img image.Image) ([]string, error) {
	// Extract dominant colors using k-means clustering
	dominantColors, err := prominentcolor.Kmeans(img)
	if err != nil {
		log.Warnf("failed to extract dominant colors: %v", err)
		return nil, err
	}

	// Map hex colors to English names and deduplicate
	uniqueNames := make(map[string]bool)
	var colorNames []string

	for i := 0; i < len(dominantColors) && len(colorNames) < 3; i++ {
		hexColor := "#" + dominantColors[i].AsString()
		name, err := mapHexToColorName(hexColor)
		if err != nil {
			log.Warnf("failed to map hex color [%s] to name: %v", hexColor, err)
			continue
		}

		// Deduplicate color names
		if !uniqueNames[name] {
			uniqueNames[name] = true
			colorNames = append(colorNames, name)
		}
	}

	return colorNames, nil
}

// loadImageFile loads an image from a file path
func loadImageFile(filePath string) (image.Image, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer DeferClose(f)

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// mapHexToColorName takes a hex string (e.g. "#BFA57E") and returns the closest English color name
func mapHexToColorName(hex string) (string, error) {
	// Normalize hex string
	hex = strings.TrimPrefix(hex, "#")
	if !strings.HasPrefix(hex, "#") {
		hex = "#" + hex
	}

	// Parse hex to color object
	c, err := colorful.Hex(hex)
	if err != nil {
		return "", err
	}

	// Find closest predefined color using CIELAB distance
	minDist := math.MaxFloat64
	nearestName := "unknown"

	for _, bucket := range predefinedColors {
		// DistanceLab is better for human perception than Euclidean RGB
		dist := c.DistanceLab(bucket.Color)

		if dist < minDist {
			minDist = dist
			nearestName = bucket.Name
		}
	}

	return nearestName, nil
}
