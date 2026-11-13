/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package fingerprint

import (
	"bytes"
	"image"
	"image/color"
	"math"

	// side-effect codec registrations beyond the stdlib trio
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// Perceptual image hashing (W12): the image twin of the text simhash —
// deterministic, zero-model, and stable under rescale/recompress. 64-bit
// pHash: box-average resample to 32×32 grayscale → 2-D DCT → the 8×8
// low-frequency block (DC excluded) → above/below-median bits.
//
// Like the simhash, the pHash only NOMINATES candidates: near-duplicate
// confirmation is a second-stage check (thumbnail pixel-diff), because
// gradients and screenshots ride surprisingly close hashes.

const (
	phashSize  = 32 // resample target, DCT input
	phashBlock = 8  // low-frequency block kept from the DCT
)

// PerceptualHash returns the 64-bit pHash of an image.
func PerceptualHash(img image.Image) uint64 {
	if img == nil {
		return 0
	}
	gray := resampleGray(img, phashSize, phashSize)
	dct := dct2D(gray)

	// 63 coefficients (skip DC), median threshold → bits
	coeffs := make([]float64, 0, phashBlock*phashBlock-1)
	for y := 0; y < phashBlock; y++ {
		for x := 0; x < phashBlock; x++ {
			if x == 0 && y == 0 {
				continue
			}
			coeffs = append(coeffs, dct[y][x])
		}
	}
	sorted := append([]float64(nil), coeffs...)
	sortFloat64s(sorted)
	median := sorted[len(sorted)/2]

	var h uint64
	bit := 0
	for y := 0; y < phashBlock; y++ {
		for x := 0; x < phashBlock; x++ {
			if x == 0 && y == 0 {
				continue
			}
			if dct[y][x] > median {
				h |= 1 << uint(bit)
			}
			bit++
		}
	}
	return h
}

// PhashHammingDistance counts differing bits between two pHashes.
func PhashHammingDistance(a, b uint64) int {
	return popcount(a ^ b)
}

// ImagePhash decodes image bytes and returns their pHash (0 when the bytes
// are not a decodable image — callers treat 0 as "no fingerprint", same as
// the text hash's !ok).
func ImagePhash(data []byte) uint64 {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return 0
	}
	return PerceptualHash(img)
}

// resampleGray box-averages the image to a w×h grayscale grid — cheaper
// than bilinear and perfectly adequate for a frequency-domain fingerprint.
func resampleGray(img image.Image, w, h int) [][]float64 {
	b := img.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	out := make([][]float64, h)
	for y := 0; y < h; y++ {
		out[y] = make([]float64, w)
	}
	if srcW == 0 || srcH == 0 {
		return out
	}
	for y := 0; y < h; y++ {
		y0 := b.Min.Y + y*srcH/h
		y1 := b.Min.Y + (y+1)*srcH/h
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < w; x++ {
			x0 := b.Min.X + x*srcW/w
			x1 := b.Min.X + (x+1)*srcW/w
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var sum float64
			var n int
			for sy := y0; sy < y1 && sy < b.Max.Y; sy++ {
				for sx := x0; sx < x1 && sx < b.Max.X; sx++ {
					sum += luminance(img.At(sx, sy))
					n++
				}
			}
			if n > 0 {
				out[y][x] = sum / float64(n)
			}
		}
	}
	return out
}

func luminance(c color.Color) float64 {
	r, g, bl, _ := c.RGBA()
	return 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)
}

// dct2D is a plain separable DCT-II over a square grid.
func dct2D(grid [][]float64) [][]float64 {
	n := len(grid)
	first := make([][]float64, n)
	for y := 0; y < n; y++ {
		first[y] = dct1D(grid[y])
	}
	// transpose, run rows again, transpose back
	t := transpose(first)
	for y := 0; y < n; y++ {
		t[y] = dct1D(t[y])
	}
	return transpose(t)
}

func dct1D(v []float64) []float64 {
	n := len(v)
	out := make([]float64, n)
	for k := 0; k < n; k++ {
		var sum float64
		for i := 0; i < n; i++ {
			sum += v[i] * math.Cos(math.Pi/float64(n)*(float64(i)+0.5)*float64(k))
		}
		out[k] = sum
	}
	return out
}

func transpose(m [][]float64) [][]float64 {
	n := len(m)
	out := make([][]float64, n)
	for i := range out {
		out[i] = make([]float64, n)
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			out[x][y] = m[y][x]
		}
	}
	return out
}

func popcount(v uint64) int {
	n := 0
	for v != 0 {
		v &= v - 1
		n++
	}
	return n
}

// sortFloat64s is an insertion sort — 63 elements, allocation-free beats
// importing sort for this.
func sortFloat64s(v []float64) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}
