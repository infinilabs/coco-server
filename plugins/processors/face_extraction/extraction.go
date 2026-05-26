/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package face_extraction

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/processors/fileproc"
)

// extractFacesAndRecognizeNames is the main orchestration for face extraction
// and recognition.  It must run AFTER text_extraction so that doc.Chunks is
// populated (used by extractSurroundingText for image files).
func (p *FaceExtractionProcessor) extractFacesAndRecognizeNames(ctx context.Context, doc *core.Document, localPath, contentType string) error {
	if err := initPigoClassifier(p.config.PigoFacefinderPath); err != nil {
		return fmt.Errorf("failed to initialize pigo classifier: %w", err)
	}

	log.Infof("[%s] starting face extraction for document [%s/%s]", p.Name(), doc.Title, doc.ID)

	surroundingTextMap, err := extractSurroundingText(ctx, p.config.TikaEndpoint, p.config.TikaTimeoutInSeconds, localPath, doc, contentType)
	if err != nil {
		return fmt.Errorf("failed to extract surrounding text: %w", err)
	}
	log.Tracef("[%s] found surrounding text for %d images", p.Name(), len(surroundingTextMap))

	tempDir, err := os.MkdirTemp("", "coco-face-work-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var imageFiles []string
	if fileproc.IsImage(localPath) {
		imageFiles = []string{localPath}
	} else {
		imageFiles, err = p.extractEmbeddedImages(ctx, localPath, tempDir)
		if err != nil {
			return fmt.Errorf("failed to extract embedded images: %w", err)
		}
	}

	if len(imageFiles) == 0 {
		log.Debugf("[%s] no images found in document [%s/%s]", p.Name(), doc.Title, doc.ID)
		return nil
	}
	log.Tracef("[%s] found %d images for processing", p.Name(), len(imageFiles))

	var allUsers []User

	for i, imgPath := range imageFiles {
		log.Tracef("[%s] processing image %d/%d: %s", p.Name(), i+1, len(imageFiles), filepath.Base(imgPath))

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
			continue
		}

		imgFileName := filepath.Base(imgPath)
		surroundingText, ok := surroundingTextMap[imgFileName]
		if !ok {
			log.Warnf("[%s] no surrounding text for image [%s], skipping", p.Name(), imgFileName)
			continue
		}

		recognitionResults, err := recognizeFacesWithAI(ctx, p, imgPath, faceImagePaths, surroundingText)
		if err != nil {
			log.Warnf("[%s] AI recognition failed for %s: %v", p.Name(), imgFileName, err)
			continue
		}

		for _, result := range recognitionResults {
			if result.Name == "" {
				continue
			}
			if result.FaceIndex < 0 || result.FaceIndex >= len(faceImagePaths) {
				log.Warnf("[%s] invalid face_index %d (valid 0-%d), skipping", p.Name(), result.FaceIndex, len(faceImagePaths)-1)
				continue
			}
			avatarBase64, err := faceImageToBase64(faceImagePaths[result.FaceIndex])
			if err != nil {
				log.Warnf("[%s] failed to convert face to base64: %v", p.Name(), err)
				avatarBase64 = ""
			}
			allUsers = append(allUsers, User{Name: result.Name, Avatar: avatarBase64})
		}
	}

	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["users"] = allUsers
	log.Infof("[%s] completed face extraction for [%s/%s]: %d user(s)", p.Name(), doc.Title, doc.ID, len(allUsers))
	return nil
}

// extractEmbeddedImages unpacks all embedded images from localPath using Tika
// and returns their paths in tempDir.
func (p *FaceExtractionProcessor) extractEmbeddedImages(ctx context.Context, localPath, tempDir string) ([]string, error) {
	if err := fileproc.TikaUnpackAllTo(ctx, p.config.TikaEndpoint, localPath, tempDir, p.config.TikaTimeoutInSeconds); err != nil {
		return nil, fmt.Errorf("tika unpack failed: %w", err)
	}

	var imageFiles []string
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && fileproc.IsImage(path) {
			imageFiles = append(imageFiles, path)
		}
		return nil
	})
	return imageFiles, err
}
