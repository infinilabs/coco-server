/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package text_attachment_extraction

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"strings"
)

// Native MHTML parsing (W2 L0.5): a MIME multipart of the page HTML plus
// its images. The largest non-ad text/html part wins (WeKnora's
// googleads/doubleclick filter); images are skipped — attachment wiring is
// the W14 scope, not this stage. Transfer encodings (base64,
// quoted-printable) are decoded per part headers.

var mhtmlAdHosts = []string{"googleads", "doubleclick", "adsystem", "adnxs"}

func parseMhtml(data []byte) ([]string, error) {
	br := bufio.NewReader(bytes.NewReader(data))
	header, err := textproto.NewReader(br).ReadMIMEHeader()
	if err != nil {
		return nil, fmt.Errorf("mhtml headers unreadable: %w", err)
	}
	ct := header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return nil, fmt.Errorf("not a multipart mhtml: %v", ct)
	}

	mr := multipart.NewReader(br, params["boundary"])

	type candidate struct {
		html string
		size int
	}
	var best candidate

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if ct := part.Header.Get("Content-Type"); !strings.Contains(strings.ToLower(ct), "text/html") {
			continue
		}
		if loc := strings.ToLower(part.Header.Get("Content-Location")); adMhtmlPart(loc) {
			continue
		}
		body, err := io.ReadAll(part)
		if err != nil {
			continue
		}
		body, err = decodeMhtmlTransfer(part.Header.Get("Content-Transfer-Encoding"), body)
		if err != nil {
			continue
		}
		if len(body) > best.size {
			text, terr := htmlToText(bytes.NewReader(body))
			if terr != nil {
				continue
			}
			best = candidate{html: text, size: len(body)}
		}
	}
	if strings.TrimSpace(best.html) == "" {
		return nil, fmt.Errorf("mhtml carries no readable html part")
	}
	return []string{best.html}, nil
}

func adMhtmlPart(location string) bool {
	for _, ad := range mhtmlAdHosts {
		if strings.Contains(location, ad) {
			return true
		}
	}
	return false
}

func decodeMhtmlTransfer(encoding string, body []byte) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		return base64.StdEncoding.DecodeString(string(body))
	case "quoted-printable":
		return io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
	default: // 7bit, 8bit, binary, empty
		return body, nil
	}
}
