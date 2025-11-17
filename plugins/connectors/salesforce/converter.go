/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package salesforce

import (
	"fmt"
	"infini.sh/coco/core"
	"strconv"
	"strings"
	"time"

	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/util"
)

// convertToDocument converts a Salesforce record to a common.Document
func convertToDocument(record map[string]interface{}, objectType string, datasource *core.DataSource, instanceUrl string) *core.Document {
	if record == nil {
		return nil
	}

	doc := &core.Document{
		Source: core.DataSourceReference{
			ID:   datasource.ID,
			Type: "connector",
			Name: datasource.Name,
		},
	}

	// Set basic fields
	doc.ID = getString(record, "Id")
	doc.Type = objectType
	doc.Icon = getIcon(objectType)

	// Set title based on object type
	doc.Title = getTitle(record, objectType)

	// Set URL - construct using instanceUrl and record Id
	if recordId := getString(record, "Id"); recordId != "" {
		doc.URL = fmt.Sprintf("%s/%s", instanceUrl, recordId)
	}

	// Set timestamps
	if createdDate := getTime(getString(record, "CreatedDate")); !createdDate.IsZero() {
		doc.Created = &createdDate
	} else {
		now := time.Now()
		doc.Created = &now
	}
	if lastModified := getTime(getString(record, "LastModifiedDate")); !lastModified.IsZero() {
		doc.Updated = &lastModified
	} else {
		doc.Updated = doc.Created
	}

	// Set content based on object type
	doc.Content = getContent(record, objectType)

	// Set summary
	doc.Summary = getSummary(record, objectType)

	// Set category and subcategory
	doc.Category = "Salesforce"
	doc.Subcategory = objectType

	// Set tags
	doc.Tags = getTags(record, objectType)

	// Set owner information
	owner := getOwner(record)
	doc.Owner = &owner

	// Set payload with all record data
	doc.Payload = record

	// Generate unique ID if not present
	if doc.ID == "" {
		doc.ID = util.MD5digest(fmt.Sprintf("%s-%s-%s", datasource.ID, objectType, doc.Title))
	}

	return doc
}

// convertToDocumentWithHierarchy converts a Salesforce record to a common.Document with proper hierarchy path
func convertToDocumentWithHierarchy(record map[string]interface{}, objectType string, datasource *core.DataSource, instanceUrl string) *core.Document {
	doc := convertToDocument(record, objectType, datasource, instanceUrl)
	if doc == nil {
		return nil
	}

	// Determine if it's a standard or custom object
	isStandard := false
	for _, stdObj := range StandardSObjects {
		if strings.EqualFold(stdObj, objectType) {
			isStandard = true
			break
		}
	}

	// Set hierarchy path based on object type
	var parentPath []string
	if isStandard {
		parentPath = []string{"Standard Objects", objectType}
	} else {
		parentPath = []string{"Custom Objects", objectType}
	}

	// Set category and categories for hierarchy
	doc.Category = common.GetFullPathForCategories(parentPath)
	doc.Categories = parentPath

	// Set system hierarchy path
	if doc.System == nil {
		doc.System = util.MapStr{}
	}
	doc.System[common.SystemHierarchyPathKey] = doc.Category

	// Add metadata about the SObject type
	if doc.Metadata == nil {
		doc.Metadata = util.MapStr{}
	}
	doc.Metadata["sobject_type"] = objectType
	doc.Metadata["is_standard"] = isStandard

	return doc
}

// getIcon returns the appropriate icon for a Salesforce object type
func getIcon(objectType string) string {
	switch objectType {
	case "Account":
		return "account"
	case "Contact":
		return "contact"
	case "Lead":
		return "lead"
	case "Opportunity":
		return "opportunity"
	case "Case":
		return "case"
	case "Campaign":
		return "campaign"
	default:
		return "default"
	}
}

// getTitle extracts the title from a Salesforce record based on object type
func getTitle(record map[string]interface{}, objectType string) string {
	switch objectType {
	case "Account":
		return getString(record, "Name")
	case "Opportunity":
		return getString(record, "Name")
	case "Contact":
		return getString(record, "Name")
	case "Lead":
		return getString(record, "Name")
	case "Campaign":
		return getString(record, "Name")
	case "Case":
		subject := getString(record, "Subject")
		if subject != "" {
			return subject
		}
		return getString(record, "CaseNumber")
	default:
		return getString(record, "Name")
	}
}

// ContentField defines a field to include in content extraction
type ContentField struct {
	FieldName string
	Label     string
	Formatter func(interface{}) string
}

// ObjectContentConfig defines content extraction configuration for each object type
var ObjectContentConfig = map[string][]ContentField{
	"Account": {
		{FieldName: "Description", Label: ""},
		{FieldName: "Website", Label: "Website"},
		{FieldName: "Type", Label: "Type"},
		{FieldName: "BillingAddress", Label: "Billing Address", Formatter: formatBillingAddress},
	},
	"Opportunity": {
		{FieldName: "Description", Label: ""},
		{FieldName: "StageName", Label: "Stage"},
	},
	"Contact": {
		{FieldName: "Description", Label: ""},
		{FieldName: "Email", Label: "Email"},
		{FieldName: "Phone", Label: "Phone"},
		{FieldName: "Title", Label: "Title"},
	},
	"Lead": {
		{FieldName: "Description", Label: ""},
		{FieldName: "Company", Label: "Company"},
		{FieldName: "Email", Label: "Email"},
		{FieldName: "Phone", Label: "Phone"},
		{FieldName: "Status", Label: "Status"},
	},
	"Campaign": {
		{FieldName: "Description", Label: ""},
		{FieldName: "Type", Label: "Type"},
		{FieldName: "Status", Label: "Status"},
		{FieldName: "IsActive", Label: "", Formatter: formatIsActive},
	},
	"Case": {
		{FieldName: "Description", Label: ""},
		{FieldName: "CaseNumber", Label: "Case Number"},
		{FieldName: "Status", Label: "Status"},
		{FieldName: "IsClosed", Label: "", Formatter: formatIsClosed},
	},
}

// formatBillingAddress formats billing address for display
func formatBillingAddress(value interface{}) string {
	if addr, ok := value.(string); ok && addr != "" {
		return "Billing Address: " + addr
	}
	return ""
}

// formatIsActive formats IsActive boolean for display
func formatIsActive(value interface{}) string {
	if isActive, ok := value.(bool); ok && isActive {
		return "Active Campaign"
	}
	return ""
}

// formatIsClosed formats IsClosed boolean for display
func formatIsClosed(value interface{}) string {
	if isClosed, ok := value.(bool); ok {
		if isClosed {
			return "Status: Closed"
		}
		return "Status: Open"
	}
	return ""
}

// getContent extracts the main content from a Salesforce record
func getContent(record map[string]interface{}, objectType string) string {
	var contentParts []string

	// Get content configuration for this object type
	config, exists := ObjectContentConfig[objectType]
	if !exists {
		// Fallback for unknown object types
		config = []ContentField{
			{FieldName: "Description", Label: ""},
		}
	}

	// Extract content based on configuration
	for _, field := range config {
		value := record[field.FieldName]
		if value == nil {
			continue
		}

		var content string
		if field.Formatter != nil {
			content = field.Formatter(value)
		} else {
			strValue := getString(record, field.FieldName)
			if strValue != "" {
				if field.Label != "" {
					content = field.Label + ": " + strValue
				} else {
					content = strValue
				}
			}
		}

		if content != "" {
			contentParts = append(contentParts, content)
		}
	}

	return strings.Join(contentParts, "\n")
}

// getSummary creates a summary from the record
func getSummary(record map[string]interface{}, objectType string) string {
	title := getTitle(record, objectType)
	description := getString(record, "Description")

	if description != "" && len(description) > 200 {
		return description[:200] + "..."
	}

	if description != "" {
		return description
	}

	return title
}

// getTags extracts tags from the record
func getTags(record map[string]interface{}, objectType string) []string {
	var tags []string

	// Add object type as tag
	tags = append(tags, objectType)

	// Add object-specific tags
	switch objectType {
	case "Account":
		if accountType := getString(record, "Type"); accountType != "" {
			tags = append(tags, accountType)
		}
		if rating := getString(record, "Rating"); rating != "" {
			tags = append(tags, "rating:"+rating)
		}

	case "Opportunity":
		if stageName := getString(record, "StageName"); stageName != "" {
			tags = append(tags, "stage:"+stageName)
		}

	case "Contact":
		if leadSource := getString(record, "LeadSource"); leadSource != "" {
			tags = append(tags, "source:"+leadSource)
		}

	case "Lead":
		if leadSource := getString(record, "LeadSource"); leadSource != "" {
			tags = append(tags, "source:"+leadSource)
		}
		if status := getString(record, "Status"); status != "" {
			tags = append(tags, "status:"+status)
		}

	case "Campaign":
		if campaignType := getString(record, "Type"); campaignType != "" {
			tags = append(tags, campaignType)
		}
		if status := getString(record, "Status"); status != "" {
			tags = append(tags, "status:"+status)
		}

	case "Case":
		if status := getString(record, "Status"); status != "" {
			tags = append(tags, "status:"+status)
		}
		if isClosed, ok := record["IsClosed"].(bool); ok {
			if isClosed {
				tags = append(tags, "closed")
			} else {
				tags = append(tags, "open")
			}
		}
	}

	return tags
}

// getOwner extracts owner information from the record
func getOwner(record map[string]interface{}) core.UserInfo {
	owner := core.UserInfo{}

	if ownerData, ok := record["Owner"].(map[string]interface{}); ok {
		owner.UserName = getString(ownerData, "Name")
		owner.UserID = getString(ownerData, "Id")
	}

	return owner
}

func getString(data map[string]interface{}, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func getTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.000+0000", // Salesforce format
		"2006-01-02T15:04:05.000Z",     // ISO 8601 with milliseconds
		"2006-01-02T15:04:05Z",         // ISO 8601 without milliseconds
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	// Fallback: numeric Unix timestamp (seconds/milliseconds/microseconds/nanoseconds)
	isDigits := true
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			isDigits = false
			break
		}
	}
	if isDigits {
		if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
			var sec int64
			var nsec int64
			switch {
			case len(s) <= 10: // seconds
				sec = ts
				nsec = 0
			case len(s) <= 13: // milliseconds
				sec = ts / 1_000
				nsec = (ts % 1_000) * int64(time.Millisecond)
			case len(s) <= 16: // microseconds
				sec = ts / 1_000_000
				nsec = (ts % 1_000_000) * int64(time.Microsecond)
			default: // nanoseconds
				sec = ts / 1_000_000_000
				nsec = ts % 1_000_000_000
			}
			return time.Unix(sec, nsec)
		}
	}
	return time.Time{}
}
