/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package fileproc

import (
	"bytes"
	"encoding/base64"
	"fmt"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"mime"
	"os"
	"path/filepath"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"github.com/disintegration/imaging"
	"github.com/tmc/langchaingo/llms"
)

// LoadLocalImageToContentPart reads an image file and converts it to a
// llms.ContentPart suitable for vision model calls.
// Images larger than 5 MB are compressed, and non-JPEG/PNG formats are
// re-encoded as JPEG for broad model compatibility.
func LoadLocalImageToContentPart(imagePath, imageContentFormat string) (llms.ContentPart, error) {
	const maxImageBytes = 5 * 1024 * 1024
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
	}

	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	mimeType := mime.TypeByExtension(filepath.Ext(imagePath))
	if mimeType == "" {
		return nil, fmt.Errorf("unknown MIME type for image extension: %s", filepath.Ext(imagePath))
	}

	if len(imageBytes) > maxImageBytes {
		compressed, err := compressImage(imageBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to compress image [%s]: %w", imagePath, err)
		}
		mimeType = "image/jpeg"
		imageBytes = compressed
	}

	if !allowedTypes[mimeType] {
		img, err := imaging.Decode(bytes.NewReader(imageBytes))
		if err != nil {
			return nil, fmt.Errorf("decode failed: %w", err)
		}
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 75}); err != nil {
			return nil, fmt.Errorf("encode jpeg failed: %w", err)
		}
		mimeType = "image/jpeg"
		imageBytes = buf.Bytes()
	}

	switch imageContentFormat {
	case "data_uri":
		base64Str := base64.StdEncoding.EncodeToString(imageBytes)
		dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str)
		return llms.ImageURLPart(dataURI), nil
	case "binary":
		return llms.BinaryPart(mimeType, imageBytes), nil
	default:
		panic(fmt.Sprintf("unknown [image_content_format] value: [%s]", imageContentFormat))
	}
}

// compressImage resizes the image so that neither dimension exceeds 1024 px,
// then re-encodes as JPEG at quality 75.
func compressImage(data []byte) ([]byte, error) {
	const dimensionMax = 1024
	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() > dimensionMax || bounds.Dy() > dimensionMax {
		img = imaging.Fit(img, dimensionMax, dimensionMax, imaging.Lanczos)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 75}); err != nil {
		return nil, fmt.Errorf("encode jpeg failed: %w", err)
	}
	return buf.Bytes(), nil
}
