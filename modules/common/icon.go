package common

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

func ParseAndGetIcon(connector *core.Connector, icon string) string {
	appCfg := AppConfig()

	icon = internalGetIcon(&appCfg, connector, icon)
	if icon != "" {
		icon = ConvertIconToBase64(&appCfg, icon)
	}
	return icon
}

func internalGetIcon(appCfg *core.Config, connector *core.Connector, icon string) string {
	link, ok := connector.Assets.Icons[icon]
	if ok {
		return AutoGetFullIconURL(appCfg, link)
	} else if appCfg.ServerInfo.Endpoint != "" {
		return AutoGetFullIconURL(appCfg, icon)
	}
	return icon
}

func AutoGetFullIconURL(appCfg *core.Config, icon string) string {
	baseEndpoint := appCfg.ServerInfo.Endpoint
	if util.PrefixStr(icon, "/") && baseEndpoint != "" {
		link, err := url.JoinPath(baseEndpoint, icon)
		if err == nil && link != "" {
			return link
		}
	}
	return icon
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
