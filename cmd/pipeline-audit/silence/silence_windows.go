//go:build windows

/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infinilabs.com */

// Package silence: no fd-level redirect on windows. Framework logs may
// interleave with the report — run under a shell filter there.
package silence

// Begin is a no-op on windows.
func Begin() {}

// End is a no-op on windows.
func End() {}
