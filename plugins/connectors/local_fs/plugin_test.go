/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package local_fs

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestDeduplicatePaths(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
		desc     string
	}{
		{
			name:     "empty_paths",
			input:    []string{},
			expected: []string{},
			desc:     "Empty input should return empty output",
		},
		{
			name:     "single_path",
			input:    []string{"/home/user"},
			expected: []string{"/home/user"},
			desc:     "Single path should be returned as-is",
		},
		{
			name:     "exact_duplicates",
			input:    []string{"/home/user", "/home/user", "/home/user"},
			expected: []string{"/home/user"},
			desc:     "Exact duplicate paths should be deduplicated",
		},
		{
			name:     "ancestor_descendant_linux",
			input:    []string{"/home/user", "/home/user/documents", "/home/user/documents/projects"},
			expected: []string{"/home/user"},
			desc:     "Linux paths: descendant paths should be removed when ancestor exists",
		},
		{
			name:     "no_relationship",
			input:    []string{"/home/user", "/var/log", "/tmp"},
			expected: []string{"/home/user", "/tmp", "/var/log"}, // sorted
			desc:     "Independent paths should all be kept",
		},
		{
			name:     "mixed_duplicates_and_descendants",
			input:    []string{"/home/user", "/home/user", "/home/user/documents", "/var/log", "/home/user/documents/projects"},
			expected: []string{"/home/user", "/var/log"},
			desc:     "Mixed duplicates and descendants should be properly handled",
		},
		{
			name:     "relative_paths",
			input:    []string{"./documents", "documents", "./documents/projects"},
			expected: []string{"documents"},
			desc:     "Relative paths should be cleaned and deduplicated",
		},
		{
			name:     "path_with_dots",
			input:    []string{"/home/user/../user/documents", "/home/user/documents"},
			expected: []string{"/home/user/documents"},
			desc:     "Paths with .. should be cleaned and deduplicated",
		},
		{
			name:     "trailing_slashes",
			input:    []string{"/home/user/", "/home/user", "/home/user/documents/"},
			expected: []string{"/home/user"},
			desc:     "Trailing slashes should be handled correctly",
		},
		{
			name:     "root_and_descendants",
			input:    []string{"/", "/home", "/home/user", "/var"},
			expected: []string{"/"},
			desc:     "Root path should cover all other paths",
		},
	}

	// Add Windows-specific tests if running on Windows or explicitly testing cross-platform
	// Note: On non-Windows platforms, these test Windows-style path parsing
	if runtime.GOOS == "windows" || !testing.Short() {
		windowsTests := []struct {
			name     string
			input    []string
			expected func() []string // Function to return platform-specific expected results
			desc     string
		}{
			{
				name:  "windows_paths",
				input: []string{`C:\Users\John`, `C:\Users\John\Documents`, `C:\Users\John\Documents\Projects`},
				expected: func() []string {
					if runtime.GOOS == "windows" {
						return []string{`C:\Users\John`}
					}
					// On non-Windows, backslashes are literal characters, so paths won't be seen as related
					return []string{`C:\Users\John`, `C:\Users\John\Documents`, `C:\Users\John\Documents\Projects`}
				},
				desc: "Windows paths: descendant paths should be removed on Windows",
			},
			{
				name:  "windows_mixed_slashes",
				input: []string{`C:\Users\John`, `C:/Users/John/Documents`},
				expected: func() []string {
					if runtime.GOOS == "windows" {
						return []string{`C:\Users\John`}
					}
					// On non-Windows, these are seen as completely different paths
					return []string{`C:/Users/John/Documents`, `C:\Users\John`}
				},
				desc: "Windows paths with mixed slashes should be normalized on Windows",
			},
			{
				name:  "windows_different_drives",
				input: []string{`C:\Users`, `D:\Data`, `C:\Users\John`},
				expected: func() []string {
					if runtime.GOOS == "windows" {
						return []string{`C:\Users`, `D:\Data`}
					}
					// On non-Windows, these are all independent paths
					return []string{`C:\Users`, `C:\Users\John`, `D:\Data`}
				},
				desc: "Different Windows drives should be kept separate",
			},
			{
				name:  "windows_unc_paths",
				input: []string{`\\server\share`, `\\server\share\folder`, `C:\local`},
				expected: func() []string {
					if runtime.GOOS == "windows" {
						return []string{`C:\local`, `\\server\share`}
					}
					// On non-Windows, backslashes are literal, so no relationship detected
					return []string{`C:\local`, `\\server\share`, `\\server\share\folder`}
				},
				desc: "Windows UNC paths should be handled correctly on Windows",
			},
		}

		// Convert windowsTests to regular test format and add to tests
		for _, windowsTest := range windowsTests {
			test := struct {
				name     string
				input    []string
				expected []string
				desc     string
			}{
				name:     windowsTest.name,
				input:    windowsTest.input,
				expected: windowsTest.expected(),
				desc:     windowsTest.desc,
			}
			tests = append(tests, test)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicatePaths(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("deduplicatePaths() = %v, expected %v\nDescription: %s", result, tt.expected, tt.desc)
			}
		})
	}
}

func TestDeduplicatePathsCrossPlatform(t *testing.T) {
	// Test platform-specific behavior
	tests := []struct {
		name         string
		input        []string
		expectedWin  []string
		expectedUnix []string
		desc         string
	}{
		{
			name:         "mixed_separators",
			input:        []string{"home/user", "home\\user\\documents"},
			expectedWin:  []string{"home/user"},                          // Windows treats both as same tree
			expectedUnix: []string{"home/user", "home\\user\\documents"}, // Unix treats \ as literal character
			desc:         "Mixed path separators should be handled per OS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicatePaths(tt.input)
			var expected []string
			if runtime.GOOS == "windows" {
				expected = tt.expectedWin
			} else {
				expected = tt.expectedUnix
			}

			if !reflect.DeepEqual(result, expected) {
				t.Errorf("deduplicatePaths() on %s = %v, expected %v\nDescription: %s",
					runtime.GOOS, result, expected, tt.desc)
			}
		})
	}
}

func TestIsPathDescendant(t *testing.T) {
	tests := []struct {
		name       string
		childPath  string
		parentPath string
		expected   bool
		desc       string
	}{
		{
			name:       "direct_child",
			childPath:  "/home/user/documents",
			parentPath: "/home/user",
			expected:   true,
			desc:       "Direct child should be detected",
		},
		{
			name:       "nested_child",
			childPath:  "/home/user/documents/projects/myproject",
			parentPath: "/home/user",
			expected:   true,
			desc:       "Nested child should be detected",
		},
		{
			name:       "same_path",
			childPath:  "/home/user",
			parentPath: "/home/user",
			expected:   false,
			desc:       "Same path should not be considered descendant",
		},
		{
			name:       "sibling_paths",
			childPath:  "/home/user",
			parentPath: "/home/guest",
			expected:   false,
			desc:       "Sibling paths should not be descendants",
		},
		{
			name:       "parent_child_reversed",
			childPath:  "/home",
			parentPath: "/home/user",
			expected:   false,
			desc:       "Parent should not be descendant of child",
		},
		{
			name:       "completely_different",
			childPath:  "/var/log",
			parentPath: "/home/user",
			expected:   false,
			desc:       "Completely different paths should not be descendants",
		},
		{
			name:       "root_parent",
			childPath:  "/home/user",
			parentPath: "/",
			expected:   true,
			desc:       "Everything should be descendant of root",
		},
	}

	// Add Windows-specific tests
	if runtime.GOOS == "windows" || !testing.Short() {
		windowsTests := []struct {
			name       string
			childPath  string
			parentPath string
			expected   func() bool
			desc       string
		}{
			{
				name:       "windows_direct_child",
				childPath:  `C:\Users\John\Documents`,
				parentPath: `C:\Users\John`,
				expected:   func() bool { return runtime.GOOS == "windows" },
				desc:       "Windows direct child should be detected on Windows",
			},
			{
				name:       "windows_different_drives",
				childPath:  `D:\Data`,
				parentPath: `C:\Users`,
				expected:   func() bool { return false }, // Always false regardless of platform
				desc:       "Different Windows drives should not be descendants",
			},
			{
				name:       "windows_mixed_separators",
				childPath:  `C:\Users\John\Documents`,
				parentPath: `C:/Users/John`,
				expected:   func() bool { return runtime.GOOS == "windows" },
				desc:       "Windows mixed separators should work on Windows",
			},
		}

		// Convert windowsTests to regular test format and add to tests
		for _, windowsTest := range windowsTests {
			test := struct {
				name       string
				childPath  string
				parentPath string
				expected   bool
				desc       string
			}{
				name:       windowsTest.name,
				childPath:  windowsTest.childPath,
				parentPath: windowsTest.parentPath,
				expected:   windowsTest.expected(),
				desc:       windowsTest.desc,
			}
			tests = append(tests, test)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean paths to match what the function expects
			childPath := filepath.Clean(tt.childPath)
			parentPath := filepath.Clean(tt.parentPath)

			result := isPathDescendant(childPath, parentPath)
			if result != tt.expected {
				t.Errorf("isPathDescendant(%q, %q) = %v, expected %v\nDescription: %s",
					childPath, parentPath, result, tt.expected, tt.desc)
			}
		})
	}
}
