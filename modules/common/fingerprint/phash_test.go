/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package fingerprint

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gradient renders a deterministic diagonal gradient at the given size —
// the same scene at any resolution.
func gradient(size int, invert bool) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			v := uint8((x + y) * 255 / (2 * size))
			if invert {
				v = 255 - v
			}
			img.Set(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return img
}

func pngBytes(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestPhashStableUnderRescale(t *testing.T) {
	small := PerceptualHash(gradient(64, false))
	big := PerceptualHash(gradient(128, false)) // same scene, 4× resolution
	require.NotZero(t, small)
	// bit-exact equality is not the pHash contract: box-average
	// quantization flips borderline coefficients. A pure rescale stays in
	// the low single digits — far inside the ≤8 near-duplicate band.
	assert.LessOrEqual(t, PhashHammingDistance(small, big), 4, "a pure rescale must stay in the near-duplicate band")
}

func TestPhashDistinguishesDifferentImages(t *testing.T) {
	a := PerceptualHash(gradient(64, false))
	b := PerceptualHash(gradient(64, true)) // inverted gradient
	assert.Greater(t, PhashHammingDistance(a, b), 20, "visually different images must sit far apart")

	assert.Equal(t, 0, PhashHammingDistance(a, a))
}

func TestImagePhashDecodesBytes(t *testing.T) {
	h := ImagePhash(pngBytes(t, gradient(64, false)))
	assert.NotZero(t, h)

	// bytes-to-image and image-to-hash agree
	assert.Equal(t, h, PerceptualHash(gradient(64, false)))
}

func TestImagePhashNonImageIsZero(t *testing.T) {
	assert.Zero(t, ImagePhash([]byte("definitely not an image")))
	assert.Zero(t, ImagePhash(nil))
	assert.Zero(t, PerceptualHash(nil))
}
