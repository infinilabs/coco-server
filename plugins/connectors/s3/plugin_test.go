/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package s3

import (
	"reflect"
	"testing"

	"infini.sh/coco/plugins/connectors"
)

func TestBuildParentCategoryArray(t *testing.T) {
	tests := []struct {
		name         string
		objectKey    string
		expected     []string
		desc         string
	}{
		{
			name:      "empty_object_key",
			objectKey: "",
			expected:  nil,
			desc:      "Empty object key should return nil",
		},
		{
			name:      "root_level_file",
			objectKey: "document.pdf",
			expected:  nil,
			desc:      "Root level file should return nil (no parent folders)",
		},
		{
			name:      "single_folder_file",
			objectKey: "documents/report.pdf",
			expected:  []string{"documents"},
			desc:      "File in single folder should include only the folder",
		},
		{
			name:      "nested_folders_file",
			objectKey: "documents/2023/reports/annual-report.pdf",
			expected:  []string{"documents", "2023", "reports"},
			desc:      "Deeply nested file should include all parent folders",
		},
		{
			name:      "path_with_trailing_slash",
			objectKey: "documents/subfolder/file.txt",
			expected:  []string{"documents", "subfolder"},
			desc:      "Path with trailing slash should be handled correctly",
		},
		{
			name:      "path_with_dots",
			objectKey: "documents/../documents/report.pdf",
			expected:  []string{"documents"},
			desc:      "Path with .. should be cleaned and simplified",
		},
		{
			name:      "windows_style_paths",
			objectKey: "documents\\subfolder\\file.txt",
			expected:  []string{"documents", "subfolder"},
			desc:      "Windows-style backslashes should be converted to forward slashes",
		},
		{
			name:      "complex_nested_structure",
			objectKey: "projects/web-app/src/components/Button.tsx",
			expected:  []string{"projects", "web-app", "src", "components"},
			desc:      "Complex nested project structure should preserve all levels",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := connectors.BuildParentCategoryArray(tt.objectKey)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("connectors.BuildParentCategoryArray(%q) = %v, expected %v\nDescription: %s",
					tt.objectKey, result, tt.expected, tt.desc)
			}
		})
	}
}

func TestBuildParentCategoryArrayEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		objectKey    string
		expected     []string
		desc         string
	}{
		{
			name:      "only_slashes",
			objectKey: "///",
			expected:  nil,
			desc:      "Object key with only slashes should return nil",
		},
		{
			name:      "mixed_separators",
			objectKey: "folder/subfolder\\file.txt",
			expected:  []string{"folder", "subfolder"},
			desc:      "Mixed separators should be normalized",
		},
		{
			name:      "special_characters_in_path",
			objectKey: "documents/user@domain.com/file.txt",
			expected:  []string{"documents", "user@domain.com"},
			desc:      "Special characters in folder names should be preserved",
		},
		{
			name:      "unicode_folder_names",
			objectKey: "文档/报告/2023年度报告.pdf",
			expected:  []string{"文档", "报告"},
			desc:      "Unicode characters in folder names should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := connectors.BuildParentCategoryArray(tt.objectKey)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("connectors.BuildParentCategoryArray(%q) = %v, expected %v\nDescription: %s",
					tt.objectKey, result, tt.expected, tt.desc)
			}
		})
	}
}

func TestMarkParentFoldersAsValid(t *testing.T) {
	tests := []struct {
		name             string
		objectKey        string
		expectedFolders  []string
		desc             string
	}{
		{
			name:            "root_level_file",
			objectKey:       "document.pdf",
			expectedFolders: []string{},
			desc:            "Root level file should not create any folders",
		},
		{
			name:            "single_folder_file",
			objectKey:       "documents/report.pdf",
			expectedFolders: []string{"documents"},
			desc:            "Single folder file should create one folder",
		},
		{
			name:            "nested_folders_file",
			objectKey:       "documents/2023/reports/annual-report.pdf",
			expectedFolders: []string{"documents", "documents/2023", "documents/2023/reports"},
			desc:            "Nested folder file should create all parent folders",
		},
		{
			name:            "complex_nested_structure",
			objectKey:       "projects/web-app/src/components/Button.tsx",
			expectedFolders: []string{"projects", "projects/web-app", "projects/web-app/src", "projects/web-app/src/components"},
			desc:            "Complex nested structure should create all intermediate folders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foldersWithMatchingFiles := make(map[string]bool)

			connectors.MarkParentFoldersAsValid(tt.objectKey, foldersWithMatchingFiles)

			// Check that all expected folders are marked
			for _, expectedFolder := range tt.expectedFolders {
				if !foldersWithMatchingFiles[expectedFolder] {
					t.Errorf("Expected folder %q to be marked as valid, but it wasn't", expectedFolder)
				}
			}

			// Check that no unexpected folders are marked
			if len(foldersWithMatchingFiles) != len(tt.expectedFolders) {
				var actualFolders []string
				for folder := range foldersWithMatchingFiles {
					actualFolders = append(actualFolders, folder)
				}
				t.Errorf("Expected %d folders %v, but got %d folders %v\nDescription: %s",
					len(tt.expectedFolders), tt.expectedFolders, len(foldersWithMatchingFiles), actualFolders, tt.desc)
			}
		})
	}
}