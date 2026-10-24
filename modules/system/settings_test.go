/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	"strings"
	"testing"

	"infini.sh/coco/core"
)

func TestValidateDocumentProcessingTika(t *testing.T) {
	valid := []core.DocumentProcessing{
		{},
		{TikaEndpoint: "http://127.0.0.1:9998"},
		{TikaEndpoint: "https://tika.internal:9998/"},
		{TikaEndpoint: "http://127.0.0.1:9998", TikaTimeoutInSeconds: 360},
		{TikaTimeoutInSeconds: 1},
		{TikaTimeoutInSeconds: 3600},
	}
	for _, cfg := range valid {
		if err := validateDocumentProcessingTika(&cfg); err != nil {
			t.Errorf("expected %+v to pass, got %v", cfg, err)
		}
	}

	invalid := []struct {
		cfg    core.DocumentProcessing
		reason string
	}{
		{core.DocumentProcessing{TikaEndpoint: "127.0.0.1:9998"}, "missing scheme"},
		{core.DocumentProcessing{TikaEndpoint: "ftp://tika:9998"}, "non-http scheme"},
		{core.DocumentProcessing{TikaEndpoint: "http://"}, "missing host"},
		{core.DocumentProcessing{TikaEndpoint: "://bad"}, "unparseable"},
		{core.DocumentProcessing{TikaTimeoutInSeconds: -5}, "negative timeout"},
		{core.DocumentProcessing{TikaTimeoutInSeconds: 3601}, "timeout over one hour"},
	}
	for _, c := range invalid {
		err := validateDocumentProcessingTika(&c.cfg)
		if err == nil {
			t.Errorf("expected %+v to fail (%s), got nil", c.cfg, c.reason)
			continue
		}
		if !strings.Contains(err.Error(), "tika") {
			t.Errorf("expected error for %+v (%s) to mention tika, got %q", c.cfg, c.reason, err)
		}
	}
}

func TestValidateAppearance(t *testing.T) {
	const tinyPng = "data:image/png;base64,iVBORw0KGgo="

	valid := []core.AppearanceSettings{
		{},
		{Title: "Acme AI", Slogan: "Search everything"},
		{ThemeColors: &core.AppearanceThemeColors{Primary: "#0087FF", PrimaryDark: "#4aa8ff", Success: "#52c41a"}},
		{ThemeColors: &core.AppearanceThemeColors{Light: &core.NeutralColors{Layout: "#EEF0F3", Container: "#FFF", BaseText: "#1F1F1F"}}},
		{Logo: &core.AppearanceLogo{Light: tinyPng, Dark: "https://cdn.example.com/logo-dark.svg", Icon: tinyPng}},
		{Search: &core.AppearanceSearch{
			Logo:       &core.AppearanceImagePair{Light: tinyPng},
			Background: &core.AppearanceImagePair{Light: "https://cdn.example.com/bg.jpg", Dark: tinyPng},
		}},
		{Login: &core.AppearanceLogin{BackgroundColor: "#0087FF", BackgroundImage: tinyPng}},
	}
	for _, cfg := range valid {
		if err := validateAppearance(&cfg); err != nil {
			t.Errorf("expected %+v to pass, got %v", cfg, err)
		}
	}

	invalid := []struct {
		cfg    core.AppearanceSettings
		reason string
	}{
		{core.AppearanceSettings{ThemeColors: &core.AppearanceThemeColors{Primary: "0087FF"}}, "hex without #"},
		{core.AppearanceSettings{ThemeColors: &core.AppearanceThemeColors{Primary: "blue"}}, "named color"},
		{core.AppearanceSettings{ThemeColors: &core.AppearanceThemeColors{Light: &core.NeutralColors{Layout: "rgb(1,2,3)"}}}, "rgb instead of hex"},
		{core.AppearanceSettings{Login: &core.AppearanceLogin{BackgroundColor: "#xyz"}}, "malformed hex"},
		{core.AppearanceSettings{Logo: &core.AppearanceLogo{Light: "data:text/html;base64,PHNjcmlwdD4="}}, "non-image data url"},
		{core.AppearanceSettings{Logo: &core.AppearanceLogo{Light: "javascript:alert(1)"}}, "javascript url"},
		{core.AppearanceSettings{Logo: &core.AppearanceLogo{Light: "/etc/passwd"}}, "relative path"},
		{core.AppearanceSettings{Login: &core.AppearanceLogin{BackgroundImage: "data:image/png;base64," + strings.Repeat("QUFB", 1024*1024)}}, "oversized image"},
	}
	for _, c := range invalid {
		err := validateAppearance(&c.cfg)
		if err == nil {
			t.Errorf("expected %+v to fail (%s), got nil", c.cfg, c.reason)
			continue
		}
		if !strings.Contains(err.Error(), "appearance") {
			t.Errorf("expected error for (%s) to mention appearance, got %q", c.reason, err)
		}
	}
}
