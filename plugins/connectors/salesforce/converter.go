/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package salesforce

import (
	"fmt"
	"strings"
	"time"

	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/util"
)

// convertToDocument converts a Salesforce record to a common.Document
func convertToDocument(record map[string]interface{}, objectType string, datasource *common.DataSource, instanceUrl string) *common.Document {
	if record == nil {
		return nil
	}

	doc := &common.Document{
		Source: common.DataSourceReference{
			ID:   datasource.ID,
			Type: "connector",
			Name: datasource.Name,
		},
	}

	// Set basic fields
	doc.ID = getStringValue(record, "Id")
	doc.Type = objectType
	doc.Icon = objectType

	// Set title based on object type
	doc.Title = getTitle(record, objectType)

	// Set URL - construct using instanceUrl and record Id
	if recordId := getStringValue(record, "Id"); recordId != "" {
		doc.URL = fmt.Sprintf("%s/%s", instanceUrl, recordId)
	}

	// Set timestamps
	if createdDate := getTimeValue(record, "CreatedDate"); createdDate != nil {
		doc.Created = createdDate
	}
	if lastModified := getTimeValue(record, "LastModifiedDate"); lastModified != nil {
		doc.Updated = lastModified
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

// getTitle extracts the title from a Salesforce record based on object type
func getTitle(record map[string]interface{}, objectType string) string {
	switch objectType {
	case "Account":
		return getStringValue(record, "Name")
	case "Opportunity":
		return getStringValue(record, "Name")
	case "Contact":
		return getStringValue(record, "Name")
	case "Lead":
		return getStringValue(record, "Name")
	case "Campaign":
		return getStringValue(record, "Name")
	case "Case":
		subject := getStringValue(record, "Subject")
		if subject != "" {
			return subject
		}
		return getStringValue(record, "CaseNumber")
	default:
		return getStringValue(record, "Name")
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
			strValue := getStringValue(record, field.FieldName)
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
	description := getStringValue(record, "Description")

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
		if accountType := getStringValue(record, "Type"); accountType != "" {
			tags = append(tags, accountType)
		}
		if rating := getStringValue(record, "Rating"); rating != "" {
			tags = append(tags, "rating:"+rating)
		}

	case "Opportunity":
		if stageName := getStringValue(record, "StageName"); stageName != "" {
			tags = append(tags, "stage:"+stageName)
		}

	case "Contact":
		if leadSource := getStringValue(record, "LeadSource"); leadSource != "" {
			tags = append(tags, "source:"+leadSource)
		}

	case "Lead":
		if leadSource := getStringValue(record, "LeadSource"); leadSource != "" {
			tags = append(tags, "source:"+leadSource)
		}
		if status := getStringValue(record, "Status"); status != "" {
			tags = append(tags, "status:"+status)
		}

	case "Campaign":
		if campaignType := getStringValue(record, "Type"); campaignType != "" {
			tags = append(tags, campaignType)
		}
		if status := getStringValue(record, "Status"); status != "" {
			tags = append(tags, "status:"+status)
		}

	case "Case":
		if status := getStringValue(record, "Status"); status != "" {
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
func getOwner(record map[string]interface{}) common.UserInfo {
	owner := common.UserInfo{}

	if ownerData, ok := record["Owner"].(map[string]interface{}); ok {
		owner.UserName = getStringValue(ownerData, "Name")
		owner.UserID = getStringValue(ownerData, "Id")
	}

	return owner
}

// getBillingAddress formats billing address information
func getBillingAddress(record map[string]interface{}) string {
	if billingAddr, ok := record["BillingAddress"].(map[string]interface{}); ok {
		var parts []string

		if street := getStringValue(billingAddr, "street"); street != "" {
			parts = append(parts, street)
		}
		if city := getStringValue(billingAddr, "city"); city != "" {
			parts = append(parts, city)
		}
		if state := getStringValue(billingAddr, "state"); state != "" {
			parts = append(parts, state)
		}
		if postalCode := getStringValue(billingAddr, "postalCode"); postalCode != "" {
			parts = append(parts, postalCode)
		}
		if country := getStringValue(billingAddr, "country"); country != "" {
			parts = append(parts, country)
		}

		return strings.Join(parts, ", ")
	}

	return ""
}

// Helper functions for extracting values from maps

func getStringValue(data map[string]interface{}, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func getTimeValue(data map[string]interface{}, key string) *time.Time {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			if t, err := time.Parse(time.RFC3339, str); err == nil {
				return &t
			}
		}
	}
	return nil
}

func getBoolValue(data map[string]interface{}, key string) bool {
	if value, ok := data[key]; ok {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

func getIntValue(data map[string]interface{}, key string) int64 {
	if value, ok := data[key]; ok {
		switch v := value.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return 0
}
