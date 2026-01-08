/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"image"
	"image/color"
	"testing"
)

func TestMapHexToColorName(t *testing.T) {
	tests := []struct {
		name     string
		hex      string
		expected string
	}{
		// Primary colors
		{"pure red", "#FF0000", "red"},
		{"pure green", "#00FF00", "green"},
		{"pure blue", "#0000FF", "blue"},

		// Secondary colors
		{"pure yellow", "#FFFF00", "yellow"},
		{"pure cyan", "#00FFFF", "cyan"},

		// Neutral colors
		{"pure white", "#FFFFFF", "white"},
		{"pure black", "#000000", "black"},
		{"middle gray", "#808080", "gray"},

		// Named colors with slight variations
		{"orange-ish", "#FFA500", "orange"},
		{"pink-ish", "#FFB6C1", "pink"},
		{"purple-ish", "#800080", "purple"},
		{"brown-ish", "#8B4513", "brown"},

		// Edge cases - document actual LAB distance behavior
		// Dark red maps to brown (closer in LAB space due to low luminance)
		{"dark red", "#8B0000", "brown"},
		// Light blue maps to white (high luminance dominates in LAB)
		{"light blue", "#ADD8E6", "white"},
		// Dark green maps to gray (low luminance, low saturation in LAB)
		{"dark green", "#006400", "gray"},
		// Navy blue maps to purple (blue+low luminance in LAB space)
		{"navy blue", "#000080", "purple"},

		// Hex format variations
		{"lowercase hex", "#ff0000", "red"},
		{"no hash prefix", "00FF00", "green"},
		{"mixed case", "#FfFf00", "yellow"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := mapHexToColorName(tc.hex)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tc.expected {
				t.Errorf("mapHexToColorName(%q) = %q, expected %q", tc.hex, result, tc.expected)
			}
		})
	}
}

func TestMapHexToColorName_InvalidHex(t *testing.T) {
	invalidHexCodes := []string{
		"not-a-color",
		"#GGG",
		"",
	}

	for _, hex := range invalidHexCodes {
		t.Run(hex, func(t *testing.T) {
			_, err := mapHexToColorName(hex)
			if err == nil {
				t.Errorf("expected error for invalid hex %q, got nil", hex)
			}
		})
	}
}

func TestExtractDominantColors_SolidColorImage(t *testing.T) {
	// Create a solid red image
	img := createSolidColorImage(100, 100, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	colors, err := ExtractDominantColors(img)
	if err != nil {
		t.Fatalf("ExtractDominantColors failed: %v", err)
	}

	if len(colors) == 0 {
		t.Fatal("expected at least one color, got none")
	}

	// A solid red image should return "red" as dominant color
	if colors[0] != "red" {
		t.Errorf("expected first color to be 'red', got %q", colors[0])
	}
}

func TestExtractDominantColors_MultiColorImage(t *testing.T) {
	// Create an image with distinct color regions
	img := createMultiColorImage(300, 100)

	colors, err := ExtractDominantColors(img)
	if err != nil {
		t.Fatalf("ExtractDominantColors failed: %v", err)
	}

	// Should return up to 3 unique colors
	if len(colors) > 3 {
		t.Errorf("expected at most 3 colors, got %d", len(colors))
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, c := range colors {
		if seen[c] {
			t.Errorf("duplicate color found: %s", c)
		}
		seen[c] = true
	}
}

// Helper functions for creating test images
//
// Generate an image with 1 color
func createSolidColorImage(width, height int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// Helper functions for creating test images
//
// Generate an image with 3 color (red, green, blue)
func createMultiColorImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	sectionWidth := width / 3

	colors := []color.Color{
		color.RGBA{R: 255, G: 0, B: 0, A: 255}, // red
		color.RGBA{R: 0, G: 255, B: 0, A: 255}, // green
		color.RGBA{R: 0, G: 0, B: 255, A: 255}, // blue
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			section := x / sectionWidth
			if section >= len(colors) {
				section = len(colors) - 1
			}
			img.Set(x, y, colors[section])
		}
	}
	return img
}
