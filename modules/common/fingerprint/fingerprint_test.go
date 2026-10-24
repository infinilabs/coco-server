/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package fingerprint

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"  Hello   World  ", "hello world"},
		{"ＡＢＣ１２３", "abc123"},     // full-width folds to half-width
		{"ｆｏｏ　ｂａｒ", "foo bar"},   // full-width space
		{"ｶﾀｶﾅ", "カタカナ"},         // half-width katakana folds to full-width
		{"\n\t\rMix\r\n", "mix"}, // all whitespace collapses
	}
	for _, c := range cases {
		if got := Normalize(c.in); got != c.want {
			t.Fatalf("normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCompute(t *testing.T) {
	// deterministic: same text, same pair
	h1, s1, ok := Compute("支付服务运维手册，涵盖部署、扩容与故障排查。")
	h2, s2, ok2 := Compute("支付服务运维手册，涵盖部署、扩容与故障排查。")
	if !ok || !ok2 || h1 != h2 || s1 != s2 {
		t.Fatalf("fingerprint not deterministic: (%s,%d,%v) vs (%s,%d,%v)", h1, s1, ok, h2, s2, ok2)
	}
	// normalization folds presentation differences into one fingerprint
	hf, _, _ := Compute("Ｐａｙｍｅｎｔ　Ｓｅｒｖｉｃｅ　Ｏｐｓ　Ｍａｎｕａｌ　ｖ２")
	hg, _, _ := Compute("Payment Service Ops Manual v2")
	if hf == "" || hf != hg {
		t.Fatalf("full-width and half-width texts must share a fingerprint: %s vs %s", hf, hg)
	}
	// a small edit flips only a few simhash bits — near, not equal
	_, sa, _ := Compute("年终双薪发放规则与计算示例")
	_, sb, _ := Compute("年终双薪发放规则与计算例示")
	if sa == sb {
		t.Fatalf("distinct texts must not collide outright")
	}
	// short text is rejected so callers clear stale fields
	if h, _, ok := Compute("短文"); ok || h != "" {
		t.Fatalf("short text must not fingerprint: %s %v", h, ok)
	}
}
