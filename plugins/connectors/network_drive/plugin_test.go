/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package network_drive

import (
	"infini.sh/coco/core"
	"reflect"
	"testing"

	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
)

func TestBuildParentCategoryArray(t *testing.T) {

	tests := []struct {
		name     string
		filePath string
		expected []string
		desc     string
	}{
		{
			name:     "empty_path",
			filePath: "",
			expected: nil,
			desc:     "Empty file path should return nil",
		},
		{
			name:     "root_level_file",
			filePath: "document.pdf",
			expected: nil,
			desc:     "Root level file should return nil (no parent folders)",
		},
		{
			name:     "single_folder_file",
			filePath: "documents/report.pdf",
			expected: []string{"documents"},
			desc:     "File in single folder should include only the folder",
		},
		{
			name:     "nested_folders_file",
			filePath: "documents/2023/reports/annual-report.pdf",
			expected: []string{"documents", "2023", "reports"},
			desc:     "Deeply nested file should include all parent folders",
		},
		{
			name:     "windows_style_paths",
			filePath: "documents\\subfolder\\file.txt",
			expected: []string{"documents", "subfolder"},
			desc:     "Windows-style backslashes should be converted to forward slashes",
		},
		{
			name:     "complex_nested_structure",
			filePath: "shared/projects/web-app/src/components/Button.tsx",
			expected: []string{"shared", "projects", "web-app", "src", "components"},
			desc:     "Complex nested project structure should preserve all levels",
		},
		{
			name:     "path_with_dots",
			filePath: "documents/../documents/report.pdf",
			expected: []string{"documents"},
			desc:     "Path with .. should be cleaned and simplified",
		},
		{
			name:     "network_share_path",
			filePath: "Users/john.doe/Documents/project.docx",
			expected: []string{"Users", "john.doe", "Documents"},
			desc:     "Network share user path should preserve user folder hierarchy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := connectors.BuildParentCategoryArray(tt.filePath)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("connectors.BuildParentCategoryArray(%q) = %v, expected %v\nDescription: %s",
					tt.filePath, result, tt.expected, tt.desc)
			}
		})
	}
}

func TestMarkParentFoldersAsValid(t *testing.T) {
	tests := []struct {
		name            string
		filePath        string
		expectedFolders []string
		desc            string
	}{
		{
			name:            "root_level_file",
			filePath:        "document.pdf",
			expectedFolders: []string{},
			desc:            "Root level file should not create any folders",
		},
		{
			name:            "single_folder_file",
			filePath:        "documents/report.pdf",
			expectedFolders: []string{"documents"},
			desc:            "Single folder file should create one folder",
		},
		{
			name:            "nested_folders_file",
			filePath:        "documents/2023/reports/annual-report.pdf",
			expectedFolders: []string{"documents", "documents/2023", "documents/2023/reports"},
			desc:            "Nested folder file should create all parent folders",
		},
		{
			name:            "network_share_structure",
			filePath:        "Users/john.doe/Documents/Projects/project.docx",
			expectedFolders: []string{"Users", "Users/john.doe", "Users/john.doe/Documents", "Users/john.doe/Documents/Projects"},
			desc:            "Network share structure should create all intermediate folders",
		},
		{
			name:            "windows_style_paths",
			filePath:        "shared\\department\\finance\\budget.xlsx",
			expectedFolders: []string{"shared", "shared/department", "shared/department/finance"},
			desc:            "Windows-style paths should be normalized and create proper folders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foldersWithMatchingFiles := make(map[string]bool)

			connectors.MarkParentFoldersAsValid(tt.filePath, foldersWithMatchingFiles)

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

func TestCreateDocumentWithHierarchy(t *testing.T) {
	datasource := &core.DataSource{
		Name: "Test Network Drive",
	}
	// Set the ID manually via the embedded ID field
	datasource.ID = "test-datasource"

	tests := []struct {
		name                string
		docType             string
		icon                string
		title               string
		content             string
		url                 string
		size                int
		parentCategoryArray []string
		idSuffix            string
		expectedCategory    string
		expectedParentPath  string
		desc                string
	}{
		{
			name:                "root_level_document",
			docType:             "file",
			icon:                "file",
			title:               "document.pdf",
			content:             "",
			url:                 "//server/share/document.pdf",
			size:                1024,
			parentCategoryArray: nil,
			idSuffix:            "server-share-document.pdf",
			expectedCategory:    "/",
			expectedParentPath:  "/",
			desc:                "Root level document should have category and parent_path set to '/'",
		},
		{
			name:                "nested_document",
			docType:             "file",
			icon:                "file",
			title:               "report.pdf",
			content:             "",
			url:                 "//server/share/documents/reports/report.pdf",
			size:                2048,
			parentCategoryArray: []string{"documents", "reports"},
			idSuffix:            "server-share-documents/reports/report.pdf",
			expectedCategory:    "/documents/reports/",
			expectedParentPath:  "/documents/reports/",
			desc:                "Nested document should have proper hierarchical category",
		},
		{
			name:                "folder_document",
			docType:             "folder",
			icon:                "folder",
			title:               "documents",
			content:             "",
			url:                 "//server/share/documents/",
			size:                0,
			parentCategoryArray: nil,
			idSuffix:            "server-share-folder-documents",
			expectedCategory:    "/",
			expectedParentPath:  "/",
			desc:                "First-level folder should have parent_path set to '/'",
		},
		{
			name:                "nested_folder",
			docType:             "folder",
			icon:                "folder",
			title:               "reports",
			content:             "",
			url:                 "//server/share/documents/reports/",
			size:                0,
			parentCategoryArray: []string{"documents"},
			idSuffix:            "server-share-folder-documents/reports",
			expectedCategory:    "/documents/",
			expectedParentPath:  "/documents/",
			desc:                "Nested folder should have proper parent category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := connectors.CreateDocumentWithHierarchy(tt.docType, tt.icon, tt.title,
				tt.url, tt.size, tt.parentCategoryArray, datasource, tt.idSuffix)

			// Check basic fields
			if doc.Type != tt.docType {
				t.Errorf("Expected doc.Type = %q, got %q", tt.docType, doc.Type)
			}
			if doc.Icon != tt.icon {
				t.Errorf("Expected doc.Icon = %q, got %q", tt.icon, doc.Icon)
			}
			if doc.Title != tt.title {
				t.Errorf("Expected doc.Title = %q, got %q", tt.title, doc.Title)
			}
			if doc.URL != tt.url {
				t.Errorf("Expected doc.URL = %q, got %q", tt.url, doc.URL)
			}
			if doc.Size != tt.size {
				t.Errorf("Expected doc.Size = %d, got %d", tt.size, doc.Size)
			}

			// Check hierarchy fields
			if doc.Category != tt.expectedCategory {
				t.Errorf("Expected doc.Category = %q, got %q", tt.expectedCategory, doc.Category)
			}

			parentPath, exists := doc.System[common.SystemHierarchyPathKey]
			if !exists {
				t.Errorf("Expected doc.System[%q] to exist", common.SystemHierarchyPathKey)
			} else if parentPath != tt.expectedParentPath {
				t.Errorf("Expected doc.System[%q] = %q, got %q", common.SystemHierarchyPathKey, tt.expectedParentPath, parentPath)
			}

			// Check Categories array
			if !reflect.DeepEqual(doc.Categories, tt.parentCategoryArray) {
				t.Errorf("Expected doc.Categories = %v, got %v", tt.parentCategoryArray, doc.Categories)
			}

			// Check that ID is generated
			if doc.ID == "" {
				t.Errorf("Expected doc.ID to be generated, but it's empty")
			}

			// Check datasource reference
			if doc.Source.ID != datasource.ID {
				t.Errorf("Expected doc.Source.ID = %q, got %q", datasource.ID, doc.Source.ID)
			}
		})
	}
}

func TestBuildParentCategoryArrayEdgeCases(t *testing.T) {

	tests := []struct {
		name     string
		filePath string
		expected []string
		desc     string
	}{
		{
			name:     "only_slashes",
			filePath: "///",
			expected: nil,
			desc:     "File path with only slashes should return nil",
		},
		{
			name:     "mixed_separators",
			filePath: "folder/subfolder\\file.txt",
			expected: []string{"folder", "subfolder"},
			desc:     "Mixed separators should be normalized",
		},
		{
			name:     "special_characters_in_path",
			filePath: "shared/user@domain.com/file.txt",
			expected: []string{"shared", "user@domain.com"},
			desc:     "Special characters in folder names should be preserved",
		},
		{
			name:     "unicode_folder_names",
			filePath: "共享/文档/报告/2023年度报告.pdf",
			expected: []string{"共享", "文档", "报告"},
			desc:     "Unicode characters in folder names should be preserved",
		},
		{
			name:     "long_nested_path",
			filePath: "level1/level2/level3/level4/level5/level6/file.txt",
			expected: []string{"level1", "level2", "level3", "level4", "level5", "level6"},
			desc:     "Very deeply nested paths should preserve all levels",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := connectors.BuildParentCategoryArray(tt.filePath)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("connectors.BuildParentCategoryArray(%q) = %v, expected %v\nDescription: %s",
					tt.filePath, result, tt.expected, tt.desc)
			}
		})
	}
}
