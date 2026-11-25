/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import "testing"

func TestEffectiveTikaEndpoint(t *testing.T) {
	var nilSection *DocumentProcessing
	if got := nilSection.EffectiveTikaEndpoint(); got != DefaultTikaEndpoint {
		t.Errorf("nil section: got %q, want default %q", got, DefaultTikaEndpoint)
	}

	empty := &DocumentProcessing{}
	if got := empty.EffectiveTikaEndpoint(); got != DefaultTikaEndpoint {
		t.Errorf("empty section: got %q, want default %q", got, DefaultTikaEndpoint)
	}

	blank := &DocumentProcessing{TikaEndpoint: "   "}
	if got := blank.EffectiveTikaEndpoint(); got != DefaultTikaEndpoint {
		t.Errorf("blank endpoint: got %q, want default %q", got, DefaultTikaEndpoint)
	}

	configured := &DocumentProcessing{TikaEndpoint: " http://tika:9998 "}
	if got := configured.EffectiveTikaEndpoint(); got != "http://tika:9998" {
		t.Errorf("configured endpoint: got %q, want trimmed %q", got, "http://tika:9998")
	}
}

func TestEffectiveTikaTimeoutInSeconds(t *testing.T) {
	var nilSection *DocumentProcessing
	if got := nilSection.EffectiveTikaTimeoutInSeconds(); got != DefaultTikaTimeoutInSeconds {
		t.Errorf("nil section: got %d, want default %d", got, DefaultTikaTimeoutInSeconds)
	}

	unset := &DocumentProcessing{}
	if got := unset.EffectiveTikaTimeoutInSeconds(); got != DefaultTikaTimeoutInSeconds {
		t.Errorf("unset timeout: got %d, want default %d", got, DefaultTikaTimeoutInSeconds)
	}

	negative := &DocumentProcessing{TikaTimeoutInSeconds: -1}
	if got := negative.EffectiveTikaTimeoutInSeconds(); got != DefaultTikaTimeoutInSeconds {
		t.Errorf("negative timeout: got %d, want default %d", got, DefaultTikaTimeoutInSeconds)
	}

	configured := &DocumentProcessing{TikaTimeoutInSeconds: 42}
	if got := configured.EffectiveTikaTimeoutInSeconds(); got != 42 {
		t.Errorf("configured timeout: got %d, want 42", got)
	}
}
