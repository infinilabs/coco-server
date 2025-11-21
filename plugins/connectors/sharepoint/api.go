package sharepoint

import (  
	"context"  
	"encoding/json"  
	"fmt"  
	"io"  
	"net/http"  
	"time"  
  
	log "github.com/cihub/seelog"  
	"golang.org/x/oauth2"  
	"golang.org/x/oauth2/microsoft"  
)  

type SharePointAPIClient struct {
	config      *SharePointConfig
	httpClient  *http.Client
	oauthConfig *oauth2.Config
	token       *oauth2.Token
	retryClient *RetryClient
}

func NewSharePointAPIClient(config *SharePointConfig) (*SharePointAPIClient, error) {
	client := &SharePointAPIClient{
		config: config,
	}

	// 初始化OAuth配置
	client.oauthConfig = &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint:     microsoft.AzureADEndpoint(config.TenantID),
		Scopes:       []string{"https://graph.microsoft.com/.default"},
	}

	// 初始化token
	if config.AccessToken != "" {
		client.token = &oauth2.Token{
			AccessToken:  config.AccessToken,
			RefreshToken: config.RefreshToken,
			Expiry:       config.TokenExpiry,
		}
	}

	// 初始化HTTP客户端
	ctx := context.Background()
	client.httpClient = client.oauthConfig.Client(ctx, client.token)

	// 初始化重试客户端
	client.retryClient = NewRetryClient(config.RetryConfig)

	return client, nil
}

func (c *SharePointAPIClient) GetSites(ctx context.Context) ([]SharePointSite, error) {
	url := "https://graph.microsoft.com/v1.0/sites"

	var allSites []SharePointSite
	for {
		resp, err := c.retryClient.DoWithRetry(ctx, func() (*http.Response, error) {
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return nil, err
			}
			return c.httpClient.Do(req)
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get sites: %w", err)
		}

		var response struct {
			Value    []SharePointSite `json:"value"`
			NextLink string           `json:"@odata.nextLink"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		allSites = append(allSites, response.Value...)

		if response.NextLink == "" {
			break
		}
		url = response.NextLink
	}

	return allSites, nil
}

func (c *SharePointAPIClient) GetDocumentLibraries(ctx context.Context, siteID string) ([]SharePointList, error) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/sites/%s/lists?$filter=list/template eq 'documentLibrary'", siteID)

	var allLists []SharePointList
	for {
		resp, err := c.retryClient.DoWithRetry(ctx, func() (*http.Response, error) {
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return nil, err
			}
			return c.httpClient.Do(req)
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get document libraries: %w", err)
		}

		var response struct {
			Value    []SharePointList `json:"value"`
			NextLink string           `json:"@odata.nextLink"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		allLists = append(allLists, response.Value...)

		if response.NextLink == "" {
			break
		}
		url = response.NextLink
	}

	return allLists, nil
}

func (c *SharePointAPIClient) GetItems(ctx context.Context, siteID, listID string, pageSize int) ([]SharePointItem, string, error) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/sites/%s/lists/%s/items?$expand=fields,driveItem&$top=%d", siteID, listID, pageSize)

	resp, err := c.retryClient.DoWithRetry(ctx, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		return c.httpClient.Do(req)
	})

	if err != nil {
		return nil, "", fmt.Errorf("failed to get items: %w", err)
	}
	defer resp.Body.Close()

	var response struct {
		Value    []SharePointItem `json:"value"`
		NextLink string           `json:"@odata.nextLink"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, "", fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Value, response.NextLink, nil
}

func (c *SharePointAPIClient) DownloadFile(ctx context.Context, downloadURL string) ([]byte, error) {
	resp, err := c.retryClient.DoWithRetry(ctx, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
		if err != nil {
			return nil, err
		}
		return c.httpClient.Do(req)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
