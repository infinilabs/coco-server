/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
)

// extractFacesAndRecognizeNames is the main orchestration function for face extraction and recognition
func (p *FileExtractionProcessor) extractFacesAndRecognizeNames(ctx context.Context, doc *core.Document, localPath, contentType string) error {
	// Initialize pigo classifier
	if err := initPigoClassifier(p.config.PigoFacefinderPath); err != nil {
		return fmt.Errorf("failed to initialize pigo classifier: %w", err)
	}

	log.Infof("[%s] starting face extraction for document [%s]", p.Name(), doc.Title)

	// Step 1: Extract surrounding text map for each embedded image
	surroundingTextMap, err := extractSurroundingText(ctx, p, localPath, doc, contentType)
	if err != nil {
		return fmt.Errorf("failed to extract surrounding text: %w", err)
	}
	log.Tracef("[%s] found surrounding text for %d images", p.Name(), len(surroundingTextMap))

	// Step 2: Extract all embedded images to temp directory
	tempDir, err := os.MkdirTemp("", "coco-face-extraction-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Determine image extraction strategy based on file type
	var imageFiles []string

	if isImageFile(localPath) {
		// For standalone images, use the original file
		imageFiles = []string{localPath}
	} else {
		// For documents, extract embedded images
		imageFiles, err = p.extractEmbeddedImages(ctx, localPath, tempDir)
		if err != nil {
			return fmt.Errorf("failed to extract embedded images: %w", err)
		}
	}

	if len(imageFiles) == 0 {
		log.Debugf("[%s] no images found in document [%s]", p.Name(), doc.Title)
		return nil
	}

	log.Tracef("[%s] found %d images for processing", p.Name(), len(imageFiles))

	// Step 3: Process each image
	var allUsers []User

	for i, imgPath := range imageFiles {
		log.Tracef("[%s] processing image %d/%d: %s", p.Name(), i+1, len(imageFiles), filepath.Base(imgPath))

		// Step 3a: Detect faces with pigo
		faces, err := detectFacesWithPigo(imgPath)
		if err != nil {
			log.Warnf("[%s] pigo detection failed for %s: %v", p.Name(), filepath.Base(imgPath), err)
			continue
		}

		if len(faces) == 0 {
			log.Tracef("[%s] no faces detected in %s", p.Name(), filepath.Base(imgPath))
			continue
		}

		log.Tracef("[%s] detected %d face(s) in %s", p.Name(), len(faces), filepath.Base(imgPath))

		// Step 3b: Crop faces to temp directory
		faceDir := filepath.Join(tempDir, fmt.Sprintf("faces_%d", i))
		if err := os.MkdirAll(faceDir, 0755); err != nil {
			log.Warnf("[%s] failed to create face directory: %v", p.Name(), err)
			continue
		}

		var faceImagePaths []string
		for j, face := range faces {
			facePath := filepath.Join(faceDir, fmt.Sprintf("face_%d.jpg", j))
			if err := cropFaceFromImage(imgPath, face, facePath); err != nil {
				log.Warnf("[%s] failed to crop face %d: %v", p.Name(), j, err)
				continue
			}
			faceImagePaths = append(faceImagePaths, facePath)
		}

		if len(faceImagePaths) == 0 {
			log.Tracef("[%s] no faces cropped in %s", p.Name(), filepath.Base(imgPath))
			continue
		}

		// Step 3c: Get surrounding text for this image
		surroundingText := SurroundingText{}
		imgFileName := filepath.Base(imgPath)
		if st, ok := surroundingTextMap[imgFileName]; ok {
			surroundingText = st
		} else {
			log.Warnf("[%s] skipped as there is not surroundingText for it", imgFileName)
			continue
		}

		// Step 3d: Use vision model to recognize names
		recognitionResults, err := recognizeFacesWithAI(ctx, p, imgPath, faceImagePaths, surroundingText)
		if err != nil {
			log.Warnf("[%s] AI recognition failed for %s: %v", p.Name(), imgFileName, err)
			continue
		}

		// Step 3e: Convert faces to base64 and build User array
		for _, result := range recognitionResults {
			if result.Name == "" {
				// Skip the faces that are not recognized
				continue
			}

			// Validate face_index is within bounds
			if result.FaceIndex < 0 || result.FaceIndex >= len(faceImagePaths) {
				log.Warnf("[%s] invalid face_index %d (valid range: 0-%d), skipping", p.Name(), result.FaceIndex, len(faceImagePaths)-1)
				continue
			}

			imagePath := faceImagePaths[result.FaceIndex]
			avatarBase64, err := faceImageToBase64(imagePath)
			if err != nil {
				log.Warnf("[%s] failed to convert face to base64: %v", p.Name(), err)
				avatarBase64 = ""
			}

			allUsers = append(allUsers, User{
				Name:   result.Name,
				Avatar: avatarBase64,
			})
		}
	}

	// Step 4: Store result in Document.Metadata["users"]
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}

	// For JSON serialization, we need to use a type that JSON can handle
	// Convert []User to a format that can be stored in metadata
	doc.Metadata["users"] = allUsers

	log.Infof("[%s] completed face extraction for [%s]: extracted %d user(s)", p.Name(), doc.Title, len(allUsers))

	return nil
}

// extractEmbeddedImages extracts all embedded images from a document to the temp directory
func (p *FileExtractionProcessor) extractEmbeddedImages(ctx context.Context, localPath string, tempDir string) ([]string, error) {
	// Use Tika to unpack all attachments
	if err := tikaUnpackAllTo(ctx, p.config.TikaEndpoint, localPath, tempDir, p.config.TimeoutInSeconds); err != nil {
		return nil, fmt.Errorf("tika unpack failed: %w", err)
	}

	// Find all image files
	var imageFiles []string
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isImageFile(path) {
			imageFiles = append(imageFiles, path)
		}
		return nil
	})

	return imageFiles, err
}
