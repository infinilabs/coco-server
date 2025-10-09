package sharepoint

import (
	"time"
)

type SharePointConfig struct {
	SiteURL      string `config:"site_url" json:"site_url"`
	TenantID     string `config:"tenant_id" json:"tenant_id"`
	ClientID     string `config:"client_id" json:"client_id"`
	ClientSecret string `config:"client_secret" json:"client_secret"`
	AuthMethod   string `config:"auth_method" json:"auth_method"` // oauth2, certificate, password

	// 可选配置
	IncludeLibraries []string `config:"include_libraries" json:"include_libraries"`
	ExcludeFolders   []string `config:"exclude_folders" json:"exclude_folders"`
	FileTypes        []string `config:"file_types" json:"file_types"`

	// 重试配置
	RetryConfig RetryConfig `config:"retry" json:"retry"`

	// OAuth tokens
	AccessToken  string    `config:"access_token" json:"access_token"`
	RefreshToken string    `config:"refresh_token" json:"refresh_token"`
	TokenExpiry  time.Time `config:"token_expiry" json:"token_expiry"`
}

type RetryConfig struct {
	MaxRetries    int           `config:"max_retries" json:"max_retries"`
	InitialDelay  time.Duration `config:"initial_delay" json:"initial_delay"`
	MaxDelay      time.Duration `config:"max_delay" json:"max_delay"`
	BackoffFactor float64       `config:"backoff_factor" json:"backoff_factor"`
}

type SharePointSite struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	DisplayName  string    `json:"displayName"`
	WebURL       string    `json:"webUrl"`
	CreatedBy    User      `json:"createdBy"`
	LastModified time.Time `json:"lastModifiedDateTime"`
}

type SharePointList struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	DisplayName  string    `json:"displayName"`
	WebURL       string    `json:"webUrl"`
	CreatedBy    User      `json:"createdBy"`
	LastModified time.Time `json:"lastModifiedDateTime"`
}

type SharePointItem struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	WebURL          string          `json:"webUrl"`
	Size            int64           `json:"size"`
	CreatedBy       User            `json:"createdBy"`
	LastModified    time.Time       `json:"lastModifiedDateTime"`
	File            *FileInfo       `json:"file,omitempty"`
	Folder          *FolderInfo     `json:"folder,omitempty"`
	ParentReference ParentReference `json:"parentReference"`
}

type FileInfo struct {
	MimeType string `json:"mimeType"`
	Hashes   Hashes `json:"hashes"`
}

type FolderInfo struct {
	ChildCount int `json:"childCount"`
}

type Hashes struct {
	QuickXorHash string `json:"quickXorHash"`
	SHA1Hash     string `json:"sha1Hash"`
}

type ParentReference struct {
	DriveID   string `json:"driveId"`
	DriveType string `json:"driveType"`
	ID        string `json:"id"`
	Path      string `json:"path"`
}

type User struct {
	Email       string `json:"email"`
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type SharePointResponse struct {
	Value    []interface{} `json:"value"`
	NextLink string        `json:"@odata.nextLink"`
}
