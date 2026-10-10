/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package text_attachment_extraction

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"
)

// Native EPUB parsing (W2 L0.5): a zip of XHTML chapters plus an OPF
// manifest/spine that defines reading order. One page per spine document,
// in spine order — Tika dumps an epub as undifferentiated text and loses
// the chapter boundaries the chunker could key on.

type epubContainer struct {
	Rootfiles struct {
		Rootfile []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfile"`
	} `xml:"rootfiles"`
}

type epubOPF struct {
	Metadata struct {
		Title string `xml:"title"`
	} `xml:"metadata"`
	Manifest struct {
		Item []struct {
			ID        string `xml:"id,attr"`
			HREF      string `xml:"href,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		Itemref []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

func parseEpub(filePath string) ([]string, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("not a readable epub (zip): %w", err)
	}
	defer zr.Close()

	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}

	// container.xml → OPF location
	container := epubContainer{}
	if err := decodeZipXML(files["META-INF/container.xml"], &container); err != nil {
		return nil, fmt.Errorf("epub container.xml unreadable: %w", err)
	}
	if len(container.Rootfiles.Rootfile) == 0 || container.Rootfiles.Rootfile[0].FullPath == "" {
		return nil, fmt.Errorf("epub declares no rootfile")
	}
	opfPath := container.Rootfiles.Rootfile[0].FullPath
	opf := epubOPF{}
	if err := decodeZipXML(files[opfPath], &opf); err != nil {
		return nil, fmt.Errorf("epub opf unreadable: %w", err)
	}

	// manifest id → href (resolved relative to the OPF's directory)
	byID := map[string]string{}
	for _, item := range opf.Manifest.Item {
		if strings.Contains(item.MediaType, "xhtml") || strings.Contains(item.MediaType, "html") {
			byID[item.ID] = path.Join(path.Dir(opfPath), item.HREF)
		}
	}

	// spine order → one page per chapter
	pages := make([]string, 0, len(opf.Spine.Itemref))
	for _, ref := range opf.Spine.Itemref {
		href := byID[ref.IDRef]
		if href == "" {
			continue
		}
		f := files[href]
		if f == nil {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		text, terr := htmlToText(rc)
		rc.Close()
		if terr != nil || strings.TrimSpace(text) == "" {
			continue
		}
		pages = append(pages, text)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("epub has no readable chapters")
	}

	// the DC title rides as a leading line on the first page — visible in
	// chunk text without a separate metadata channel
	if title := strings.TrimSpace(opf.Metadata.Title); title != "" {
		pages[0] = "# " + title + "\n\n" + pages[0]
	}
	return pages, nil
}

func decodeZipXML(f *zip.File, v interface{}) error {
	if f == nil {
		return fmt.Errorf("epub part missing")
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, v)
}
