/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connectors

import (
	"infini.sh/coco/core"
	"reflect"
	"testing"

	"infini.sh/coco/modules/common"
)

func TestMarkParentFoldersAsValid(t *testing.T) {
	tests := []struct {
		name            string
		filePath        string
		expectedFolders []string
		desc            string
	}{
		{
			name:            "empty_path",
			filePath:        "",
			expectedFolders: []string{},
			desc:            "Empty file path should not create any folders",
		},
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
			name:            "windows_style_paths",
			filePath:        "shared\\department\\finance\\budget.xlsx",
			expectedFolders: []string{"shared", "shared/department", "shared/department/finance"},
			desc:            "Windows-style paths should be normalized and create proper folders",
		},
		{
			name:            "mixed_separators",
			filePath:        "folder/subfolder\\file.txt",
			expectedFolders: []string{"folder", "folder/subfolder"},
			desc:            "Mixed separators should be normalized",
		},
		{
			name:            "complex_nested_structure",
			filePath:        "projects/web-app/src/components/Button.tsx",
			expectedFolders: []string{"projects", "projects/web-app", "projects/web-app/src", "projects/web-app/src/components"},
			desc:            "Complex nested structure should create all intermediate folders",
		},
		{
			name:            "special_characters_in_path",
			filePath:        "shared/user@domain.com/file.txt",
			expectedFolders: []string{"shared", "shared/user@domain.com"},
			desc:            "Special characters in folder names should be preserved",
		},
		{
			name:            "unicode_folder_names",
			filePath:        "共享/文档/报告/2023年度报告.pdf",
			expectedFolders: []string{"共享", "共享/文档", "共享/文档/报告"},
			desc:            "Unicode characters in folder names should be preserved",
		},
		{
			name:            "path_with_dots",
			filePath:        "documents/../documents/report.pdf",
			expectedFolders: []string{"documents"},
			desc:            "Path with .. should be cleaned and simplified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foldersWithMatchingFiles := make(map[string]bool)

			MarkParentFoldersAsValid(tt.filePath, foldersWithMatchingFiles)

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
			filePath: "documents\\\\subfolder\\\\file.txt",
			expected: []string{"documents", "subfolder"},
			desc:     "Windows-style backslashes should be converted to forward slashes",
		},
		{
			name:     "mixed_separators",
			filePath: "folder/subfolder\\\\file.txt",
			expected: []string{"folder", "subfolder"},
			desc:     "Mixed separators should be normalized",
		},
		{
			name:     "complex_nested_structure",
			filePath: "projects/web-app/src/components/Button.tsx",
			expected: []string{"projects", "web-app", "src", "components"},
			desc:     "Complex nested project structure should preserve all levels",
		},
		{
			name:     "path_with_dots",
			filePath: "documents/../documents/report.pdf",
			expected: []string{"documents"},
			desc:     "Path with .. should be cleaned and simplified",
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
			name:     "only_slashes",
			filePath: "///",
			expected: nil,
			desc:     "File path with only slashes should return nil",
		},
		{
			name:     "path_with_trailing_slash",
			filePath: "documents/subfolder/file.txt/",
			expected: []string{"documents", "subfolder"},
			desc:     "Path with trailing slash should be handled correctly",
		},
		{
			name:     "long_nested_path",
			filePath: "level1/level2/level3/level4/level5/level6/file.txt",
			expected: []string{"level1", "level2", "level3", "level4", "level5", "level6"},
			desc:     "Very deeply nested paths should preserve all levels",
		},
		{
			name:     "network_share_path",
			filePath: "Users/john.doe/Documents/project.docx",
			expected: []string{"Users", "john.doe", "Documents"},
			desc:     "Network share user path should preserve user folder hierarchy",
		},
		{
			name:     "s3_object_key",
			filePath: "bucket-data/uploads/images/photo.jpg",
			expected: []string{"bucket-data", "uploads", "images"},
			desc:     "S3-style object keys should be handled correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildParentCategoryArray(tt.filePath)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("BuildParentCategoryArray(%q) = %v, expected %v\\nDescription: %s",
					tt.filePath, result, tt.expected, tt.desc)
			}
		})
	}
}

func TestSetDocumentHierarchy(t *testing.T) {
	tests := []struct {
		name                string
		parentCategoryArray []string
		expectedCategory    string
		expectedParentPath  string
		expectedCategories  []string
		desc                string
	}{
		{
			name:                "empty_parent_categories",
			parentCategoryArray: nil,
			expectedCategory:    "/",
			expectedParentPath:  "/",
			expectedCategories:  nil,
			desc:                "Empty parent categories should set category and parent_path to '/'",
		},
		{
			name:                "single_parent_category",
			parentCategoryArray: []string{"documents"},
			expectedCategory:    "/documents/",
			expectedParentPath:  "/documents/",
			expectedCategories:  []string{"documents"},
			desc:                "Single parent category should create proper hierarchy",
		},
		{
			name:                "nested_parent_categories",
			parentCategoryArray: []string{"documents", "2023", "reports"},
			expectedCategory:    "/documents/2023/reports/",
			expectedParentPath:  "/documents/2023/reports/",
			expectedCategories:  []string{"documents", "2023", "reports"},
			desc:                "Nested parent categories should create full hierarchy path",
		},
		{
			name:                "deep_nested_categories",
			parentCategoryArray: []string{"projects", "web-app", "src", "components"},
			expectedCategory:    "/projects/web-app/src/components/",
			expectedParentPath:  "/projects/web-app/src/components/",
			expectedCategories:  []string{"projects", "web-app", "src", "components"},
			desc:                "Deep nested categories should preserve full hierarchy",
		},
		{
			name:                "special_characters",
			parentCategoryArray: []string{"shared", "user@domain.com"},
			expectedCategory:    "/shared/user@domain.com/",
			expectedParentPath:  "/shared/user@domain.com/",
			expectedCategories:  []string{"shared", "user@domain.com"},
			desc:                "Special characters in categories should be preserved",
		},
		{
			name:                "unicode_categories",
			parentCategoryArray: []string{"共享", "文档", "报告"},
			expectedCategory:    "/共享/文档/报告/",
			expectedParentPath:  "/共享/文档/报告/",
			expectedCategories:  []string{"共享", "文档", "报告"},
			desc:                "Unicode characters in categories should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test document
			doc := &core.Document{
				Type:  "file",
				Title: "test.txt",
			}
			// Initialize System map
			doc.System = make(map[string]interface{})

			SetDocumentHierarchy(doc, tt.parentCategoryArray)

			// Check Category field
			if doc.Category != tt.expectedCategory {
				t.Errorf("Expected doc.Category = %q, got %q", tt.expectedCategory, doc.Category)
			}

			// Check System[parent_path] field
			parentPath, exists := doc.System[common.SystemHierarchyPathKey]
			if !exists {
				t.Errorf("Expected doc.System[%q] to exist", common.SystemHierarchyPathKey)
			} else if parentPath != tt.expectedParentPath {
				t.Errorf("Expected doc.System[%q] = %q, got %q", common.SystemHierarchyPathKey, tt.expectedParentPath, parentPath)
			}

			// Check Categories array
			if !reflect.DeepEqual(doc.Categories, tt.expectedCategories) {
				t.Errorf("Expected doc.Categories = %v, got %v", tt.expectedCategories, doc.Categories)
			}
		})
	}
}

func TestCreateDocumentWithHierarchy(t *testing.T) {
	// Create a test datasource
	datasource := &core.DataSource{
		Name: "Test DataSource",
	}
	datasource.ID = "test-datasource-id"
	datasource.System = map[string]interface{}{
		"test_key": "test_value",
	}

	tests := []struct {
		name                string
		docType             string
		icon                string
		title               string
		url                 string
		size                int
		parentCategoryArray []string
		idSuffix            string
		expectedCategory    string
		expectedParentPath  string
		desc                string
	}{
		{
			name:                "root_level_file",
			docType:             "file",
			icon:                "file",
			title:               "document.pdf",
			url:                 "https://example.com/document.pdf",
			size:                1024,
			parentCategoryArray: nil,
			idSuffix:            "document.pdf",
			expectedCategory:    "/",
			expectedParentPath:  "/",
			desc:                "Root level file should have category and parent_path set to '/'",
		},
		{
			name:                "nested_file",
			docType:             "file",
			icon:                "file",
			title:               "report.pdf",
			url:                 "https://example.com/documents/reports/report.pdf",
			size:                2048,
			parentCategoryArray: []string{"documents", "reports"},
			idSuffix:            "documents/reports/report.pdf",
			expectedCategory:    "/documents/reports/",
			expectedParentPath:  "/documents/reports/",
			desc:                "Nested file should have proper hierarchical category",
		},
		{
			name:                "folder_document",
			docType:             "folder",
			icon:                "folder",
			title:               "documents",
			url:                 "https://example.com/documents/",
			size:                0,
			parentCategoryArray: nil,
			idSuffix:            "folder-documents",
			expectedCategory:    "/",
			expectedParentPath:  "/",
			desc:                "First-level folder should have parent_path set to '/'",
		},
		{
			name:                "nested_folder",
			docType:             "folder",
			icon:                "folder",
			title:               "reports",
			url:                 "https://example.com/documents/reports/",
			size:                0,
			parentCategoryArray: []string{"documents"},
			idSuffix:            "folder-documents/reports",
			expectedCategory:    "/documents/",
			expectedParentPath:  "/documents/",
			desc:                "Nested folder should have proper parent category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := CreateDocumentWithHierarchy(tt.docType, tt.icon, tt.title,
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
			if doc.Source.Type != "connector" {
				t.Errorf("Expected doc.Source.Type = \"connector\", got %q", doc.Source.Type)
			}
			if doc.Source.Name != datasource.Name {
				t.Errorf("Expected doc.Source.Name = %q, got %q", datasource.Name, doc.Source.Name)
			}

			// Check that System map contains datasource system values
			if testValue, ok := doc.System["test_key"]; !ok || testValue != "test_value" {
				t.Errorf("Expected doc.System to contain datasource.System values")
			}
		})
	}
}
