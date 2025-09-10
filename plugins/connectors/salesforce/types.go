/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package salesforce

// OAuthConfig represents the OAuth configuration for Salesforce authentication
type OAuthConfig struct {
	Domain       string `config:"domain"`
	ClientID     string `config:"client_id"`
	ClientSecret string `config:"client_secret"`
}

// Config represents the configuration for the Salesforce connector
type Config struct {
	OAuth                 OAuthConfig `config:"oauth"`
	StandardObjectsToSync []string    `config:"standard_objects_to_sync"`
	SyncCustomObjects     bool        `config:"sync_custom_objects"`
	CustomObjectsToSync   []string    `config:"custom_objects_to_sync"`
}

// QueryResponse represents a Salesforce query response
type QueryResponse struct {
	TotalSize      int64                    `json:"totalSize"`
	Done           bool                     `json:"done"`
	Records        []map[string]interface{} `json:"records"`
	NextRecordsUrl string                   `json:"nextRecordsUrl,omitempty"`
}

// TokenResponse represents a Salesforce OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	InstanceUrl  string `json:"instance_url"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// StandardSObjects Standard SObject that we care about
var StandardSObjects = []string{
	"Account",
	"Opportunity",
	"Contact",
	"Lead",
	"Campaign",
	"Case",
}

// RelevantSObjects Relevant SObjects that we care about
var RelevantSObjects = []string{
	"Account",
	"Campaign",
	"Case",
	"CaseComment",
	"CaseFeed",
	"Contact",
	"ContentDocument",
	"ContentDocumentLink",
	"ContentVersion",
	"EmailMessage",
	"FeedComment",
	"Lead",
	"Opportunity",
	"User",
}

// RelevantSObjectFields Relevant SObject fields that we care about
var RelevantSObjectFields = []string{
	"AccountId",
	"BccAddress",
	"BillingAddress",
	"Body",
	"CaseNumber",
	"CcAddress",
	"CommentBody",
	"CommentCount",
	"Company",
	"ContentSize",
	"ConvertedAccountId",
	"ConvertedContactId",
	"ConvertedDate",
	"ConvertedOpportunityId",
	"Department",
	"Description",
	"Email",
	"EndDate",
	"FileExtension",
	"FirstOpenedDate",
	"FromAddress",
	"FromName",
	"IsActive",
	"IsClosed",
	"IsDeleted",
	"LastEditById",
	"LastEditDate",
	"LastModifiedById",
	"LatestPublishedVersionId",
	"LeadSource",
	"LinkUrl",
	"MessageDate",
	"Name",
	"OwnerId",
	"ParentId",
	"Phone",
	"PhotoUrl",
	"Rating",
	"StageName",
	"StartDate",
	"Status",
	"StatusParentId",
	"Subject",
	"TextBody",
	"Title",
	"ToAddress",
	"Type",
	"VersionDataUrl",
	"VersionNumber",
	"Website",
	"UserType",
}

var TikaSupportedFiletypes = []string{
	".txt",
	".py",
	".rst",
	".html",
	".markdown",
	".json",
	".xml",
	".csv",
	".md",
	".ppt",
	".rtf",
	".docx",
	".odt",
	".xls",
	".xlsx",
	".rb",
	".paper",
	".sh",
	".pptx",
	".pdf",
	".doc",
	".aspx",
	".xlsb",
	".xlsm",
	".tsv",
	".svg",
	".msg",
	".potx",
	".vsd",
	".vsdx",
	".vsdm",
}
