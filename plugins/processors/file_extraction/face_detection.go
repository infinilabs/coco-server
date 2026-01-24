/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"sync"

	log "github.com/cihub/seelog"
	pigo "github.com/esimov/pigo/core"
)

// FaceDetection represents a detected face with bounding box
type FaceDetection struct {
	X      int
	Y      int
	Width  int
	Height int
	Score  float32
}

// SurroundingText contains text context around an embedded image
type SurroundingText struct {
	Before string
	After  string
}

// FaceRecognitionResult represents vision model output
type FaceRecognitionResult struct {
	FaceIndex int    `json:"face_index"`
	Name      string `json:"name"`
}

// User represents final output stored in Document.Metadata["users"]
type User struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"` // base64 encoded string
}

var (
	pigoClassifier     *pigo.Pigo
	pigoClassifierOnce sync.Once
	pigoInitError      error
)

// initPigoClassifier initializes the pigo classifier from the cascade file
func initPigoClassifier(pigoPath string) error {
	pigoClassifierOnce.Do(func() {
		cascadeFile, err := os.ReadFile(pigoPath)
		if err != nil {
			pigoInitError = fmt.Errorf("failed to read pigo cascade file: %w", err)
			return
		}

		pigoLib := pigo.NewPigo()
		classifier, err := pigoLib.Unpack(cascadeFile)
		if err != nil {
			pigoInitError = fmt.Errorf("failed to unpack pigo cascade file: %w", err)
			return
		}

		pigoClassifier = classifier
		log.Debugf("pigo classifier loaded from %s", pigoPath)
	})
	return pigoInitError
}

// detectFacesWithPigo uses pigo to detect faces in an image
func detectFacesWithPigo(imgPath string) ([]FaceDetection, error) {
	// Open and decode image
	file, err := os.Open(imgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Check format
	if format != "jpeg" && format != "png" && format != "gif" {
		return nil, fmt.Errorf("unsupported image format: %s", format)
	}

	// Convert to grayscale
	pixels := pigo.RgbToGrayscale(img)
	cols, rows := img.Bounds().Max.X, img.Bounds().Max.Y

	// Set detection parameters
	cParams := pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     1000,
		ShiftFactor: 0.1,
		ScaleFactor: 1.05,
		ImageParams: pigo.ImageParams{
			Pixels: pixels,
			Rows:   rows,
			Cols:   cols,
			Dim:    cols,
		},
	}

	// Run face detection
	angle := 0.0
	dets := pigoClassifier.RunCascade(cParams, angle)
	dets = pigoClassifier.ClusterDetections(dets, 0.2)

	log.Tracef("pigo detected %d faces in %s", len(dets), filepath.Base(imgPath))

	// Convert and filter results
	var faces []FaceDetection
	for _, det := range dets {
		if det.Q < 15.0 {
			continue
		}

		halfSize := det.Scale / 2
		x := int(det.Col - halfSize)
		y := int(det.Row - halfSize)
		width := int(det.Scale)
		height := int(det.Scale)

		// Boundary checks
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		if x+width > cols {
			width = cols - x
		}
		if y+height > rows {
			height = rows - y
		}

		if width <= 0 || height <= 0 {
			continue
		}

		faces = append(faces, FaceDetection{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
			Score:  det.Q,
		})
	}

	// Deduplicate faces
	faces = deduplicateFaces(faces)

	return faces, nil
}

// calculateIoU calculates Intersection over Union for two face detections
func calculateIoU(a, b FaceDetection) float32 {
	x1 := max(a.X, b.X)
	y1 := max(a.Y, b.Y)
	x2 := min(a.X+a.Width, b.X+b.Width)
	y2 := min(a.Y+a.Height, b.Y+b.Height)

	if x1 >= x2 || y1 >= y2 {
		return 0
	}

	intersection := (x2 - x1) * (y2 - y1)
	union := a.Width*a.Height + b.Width*b.Height - intersection

	if union <= 0 {
		return 0
	}

	return float32(intersection) / float32(union)
}

// deduplicateFaces removes duplicate face detections using IoU
func deduplicateFaces(faces []FaceDetection) []FaceDetection {
	if len(faces) <= 1 {
		return faces
	}

	// Sort by score (descending)
	sort.Slice(faces, func(i, j int) bool {
		return faces[i].Score > faces[j].Score
	})

	var result []FaceDetection
	used := make([]bool, len(faces))
	const iouThreshold = 0.5

	for i := 0; i < len(faces); i++ {
		if used[i] {
			continue
		}

		result = append(result, faces[i])
		used[i] = true

		for j := i + 1; j < len(faces); j++ {
			if used[j] {
				continue
			}
			if calculateIoU(faces[i], faces[j]) > iouThreshold {
				used[j] = true
			}
		}
	}

	return result
}

// cropFaceFromImage crops a face region from the original image
func cropFaceFromImage(imgPath string, face FaceDetection, outputPath string) error {
	file, err := os.Open(imgPath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	rect := image.Rect(face.X, face.Y, face.X+face.Width, face.Y+face.Height)
	rect = rect.Intersect(img.Bounds())

	croppedImg, ok := img.(interface {
		SubImage(r image.Rectangle) image.Image
	})
	if !ok {
		return fmt.Errorf("image does not support SubImage")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if err := jpeg.Encode(outFile, croppedImg.SubImage(rect), &jpeg.Options{Quality: 95}); err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	return nil
}

// faceImageToBase64 reads a face image file and returns only the base64 string
func faceImageToBase64(imagePath string) (string, error) {
	imgBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read face image: %w", err)
	}
	return base64.StdEncoding.EncodeToString(imgBytes), nil
}
