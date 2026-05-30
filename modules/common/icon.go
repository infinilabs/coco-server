package common

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

// ParseAndGetIcon does the following processing to icon:
//
//  1. Convert it to a relative URL if it is a key. Since icon could be either an
//     icon key or a absolute URL, after this step, it is guaranteed to be a URL.
//  2. If appCfg.ServerInfo.EncodeIconToBase64 is true, request the URL to fetch
//     the icon bytes, then encode them using base64, and return.
func ParseAndGetIcon(connector *core.Connector, icon string) string {
	appCfg := AppConfig()

	// The returned icno could be either relative or absolute
	icon = iconKeyToRelativeUrl(connector, icon)

	if icon != "" && appCfg.ServerInfo.EncodeIconToBase64 {
		// Convert it to the absoluteUrl URL if needed.
		absoluteUrl := AutoGetFullIconURL(&appCfg, icon)
		encoded := ConvertIconToBase64(&appCfg, absoluteUrl)
		if util.PrefixStr(encoded, "data:") {
			icon = encoded
		}
	}
	return icon
}

// If icon is an icon key (e.g., folder), convert it to a relative URL.
func iconKeyToRelativeUrl(connector *core.Connector, icon string) string {
	link, ok := connector.Assets.Icons[icon]
	if ok {
		return link
	}
	return icon
}

// Concatenate app baseEndpoint and iconUrl to generate an absolute URL.
func AutoGetFullIconURL(appCfg *core.Config, iconUrl string) string {
	baseEndpoint := appCfg.ServerInfo.Endpoint
	if util.PrefixStr(iconUrl, "/") && baseEndpoint != "" {
		link, err := url.JoinPath(baseEndpoint, iconUrl)
		if err == nil && link != "" {
			return link
		}
	}
	return iconUrl
}

func ConvertIconToBase64(appCfg *core.Config, icon string) string {
	if appCfg.ServerInfo.EncodeIconToBase64 && util.PrefixStr(icon, "http") {
		result, err := util.HttpGet(icon)
		if err == nil && result != nil {
			if result.StatusCode >= 200 && result.StatusCode < 400 && result.Body != nil {
				// Attempt to get the Content-Type from custom headers
				contentType := ""
				if ct, ok := result.Headers["Content-Type"]; ok && len(ct) > 0 {
					contentType = ct[0]
				}
				if contentType == "" {
					contentType = http.DetectContentType(result.Body)
				}
				// Encode to base64
				base64Data := base64.StdEncoding.EncodeToString(result.Body)
				icon = fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)
			}
		}
	}
	return icon
}
