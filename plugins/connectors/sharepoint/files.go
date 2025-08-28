package sharepoint  
  
import (  
	"context"  
	"fmt"  
	"path/filepath"  
	"strings"  
	"time"  
  
	log "github.com/cihub/seelog"  
	"infini.sh/coco/modules/common"  
	"infini.sh/framework/core/queue"  
	"infini.sh/framework/core/util"  
)  
  
func (p *Plugin) syncSharePointContent(connector *common.Connector, datasource *common.DataSource, config *SharePointConfig) error {  
	ctx := context.Background()  
	  
	// 获取所有站点  
	sites, err := p.apiClient.GetSites(ctx)  
	if err != nil {  
		return fmt.Errorf("failed to get sites: %w", err)  
	}  
	  
	log.Infof("[sharepoint connector] found %d sites", len(sites))  
	  
	for _, site := range sites {  
		if err := p.processSite(ctx, connector, datasource, config, site); err != nil {  
			log.Errorf("[sharepoint connector] failed to process site %s: %v", site.Name, err)  
			continue  
		}  
	}  
	  
	return nil  
}  
  
func (p *Plugin) processSite(ctx context.Context, connector *common.Connector, datasource *common.DataSource, config *SharePointConfig, site SharePointSite) error {  
	log.Debugf("[sharepoint connector] processing site: %s", site.Name)  
	  
	// 获取文档库  
	libraries, err := p.apiClient.GetDocumentLibraries(ctx, site.ID)  
	if err != nil {  
		return fmt.Errorf("failed to get document libraries: %w", err)  
	}  
	  
	for _, library := range libraries {  
		// 检查是否在包含列表中  
		if len(config.IncludeLibraries) > 0 && !contains(config.IncludeLibraries, library.Name) {  
			continue  
		}  
		  
		if err := p.processDocumentLibrary(ctx, connector, datasource, config, site, library); err != nil {  
			log.Errorf("[sharepoint connector] failed to process library %s: %v", library.Name, err)  
			continue  
		}  
	}  
	  
	return nil  
}  
  
func (p *Plugin) processDocumentLibrary(ctx context.Context, connector *common.Connector, datasource *common.DataSource, config *SharePointConfig, site SharePointSite, library SharePointList) error {  
	log.Debugf("[sharepoint connector] processing library: %s", library.Name)  
	  
	var nextLink string  
	pageSize := p.PageSize  
	if pageSize == 0 {  
		pageSize = 100 // 默认分页大小  
	}  
	  
	for {  
		items, next, err := p.apiClient.GetItems(ctx, site.ID, library.ID, pageSize)  
		if err != nil {  
			return fmt.Errorf("failed to get items: %w", err)  
		}  
		  
		for _, item := range items {  
			if err := p.processSharePointItem(ctx, connector, datasource, config, site, library, item); err != nil {  
				log.Errorf("[sharepoint connector] failed to process item %s: %v", item.Name, err)  
				continue  
			}  
		}  
		  
		nextLink = next  
		if nextLink == "" {  
			break  
		}  
	}  
	  
	return nil  
}  
  
func (p *Plugin) processSharePointItem(ctx context.Context, connector *common.Connector, datasource *common.DataSource, config *SharePointConfig, site SharePointSite, library SharePointList, item SharePointItem) error {  
	// 跳过文件夹  
	if item.Folder != nil {  
		return nil  
	}  
	  
	// 检查文件类型过滤  
	if len(config.FileTypes) > 0 {  
		ext := strings.ToLower(filepath.Ext(item.Name))  
		if !contains(config.FileTypes, ext) {  
			return nil  
		}  
	}  
	  
	// 检查排除文件夹  
	for _, excludeFolder := range config.ExcludeFolders {  
		if strings.Contains(item.ParentReference.Path, excludeFolder) {  
			return nil  
		}  
	}  
	  
	// 创建文档对象  
	document := common.Document{  
		Source: common.DataSourceReference{  
			ID:   datasource.ID,  
			Name: datasource.Name,  
			Type: "connector",  
		},  
		Title:   item.Name,  
		Type:    getFileType(item.File.MimeType),  
		Size:    int(item.Size),  
		URL:     item.WebURL,  
		Owner: &common.UserInfo{  
			UserName: item.CreatedBy.DisplayName,  
			UserID:   item.CreatedBy.Email,  
		},  
		Icon: getFileIcon(item.File.MimeType),  
	}  
	  
	// 设置系统字段  
	document.System = datasource.System  
	document.ID = util.MD5digest(fmt.Sprintf("%s-%s-%s", datasource.ID, site.ID, item.ID))  
	document.Created = &item.LastModified  
	document.Updated = &item.LastModified  
	  
	// 设置分类路径  
	categoryPath := fmt.Sprintf("/%s/%s%s", site.Name, library.Name, item.ParentReference.Path)  
	document.Category = categoryPath  
	document.Categories = strings.Split(strings.Trim(categoryPath, "/"), "/")  
	  
	// 构建富分类  
	document.RichCategories = []common.RichLabel{  
		{Label: site.Name, Key: site.ID, Icon: "site"},  
		{Label: library.Name, Key: library.ID, Icon: "library"},  
	}  

	// 尝试下载并提取文件内容  
	if shouldExtractContent(item.File.MimeType) {  
		content, err := p.extractFileContent(ctx, item)  
		if err != nil {  
			log.Warnf("[sharepoint connector] failed to extract content for %s: %v", item.Name, err)  
		} else {  
			document.Content = content  
		}  
	}  
	  
	// 推送到队列  
	p.saveDocToQueue(document)  
	  
	return nil  
}  
  
func (p *Plugin) extractFileContent(ctx context.Context, item SharePointItem) (string, error) {  
	// 构建下载URL  
	downloadURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/sites/%s/drive/items/%s/content",   
		item.ParentReference.DriveID, item.ID)  
	  
	// 下载文件内容  
	content, err := p.apiClient.DownloadFile(ctx, downloadURL)  
	if err != nil {  
		return "", err  
	}  
	  
	// 根据MIME类型提取文本内容  
	return extractTextFromBytes(content, item.File.MimeType), nil  
}  
  
func (p *Plugin) saveDocToQueue(document common.Document) {  
	data := util.MustToJSONBytes(document)  
	err := queue.Push(queue.SmartGetOrInitConfig(p.Queue), data)  
	if err != nil {  
		log.Errorf("[sharepoint connector] failed to push document to queue: %v", err)  
		panic(err)  
	}  
}  
  
// 辅助函数  
func contains(slice []string, item string) bool {  
	for _, s := range slice {  
		if s == item {  
			return true  
		}  
	}  
	return false  
}  
  
func getFileType(mimeType string) string {  
	switch {  
	case strings.HasPrefix(mimeType, "text/"):  
		return "text"  
	case strings.HasPrefix(mimeType, "image/"):  
		return "image"  
	case strings.HasPrefix(mimeType, "video/"):  
		return "video"  
	case strings.HasPrefix(mimeType, "audio/"):  
		return "audio"  
	case strings.Contains(mimeType, "pdf"):  
		return "pdf"  
	case strings.Contains(mimeType, "word"):  
		return "document"  
	case strings.Contains(mimeType, "excel"):  
		return "spreadsheet"  
	case strings.Contains(mimeType, "powerpoint"):  
		return "presentation"  
	default:  
		return "file"  
	}  
}  
  
func getFileIcon(mimeType string) string {  
	switch {  
	case strings.Contains(mimeType, "pdf"):  
		return "pdf"  
	case strings.Contains(mimeType, "word"):  
		return "word"  
	case strings.Contains(mimeType, "excel"):  
		return "excel"  
	case strings.Contains(mimeType, "powerpoint"):  
		return "powerpoint"  
	case strings.HasPrefix(mimeType, "image/"):  
		return "image"  
	case strings.HasPrefix(mimeType, "video/"):  
		return "video"  
	case strings.HasPrefix(mimeType, "audio/"):  
		return "audio"  
	default:  
		return "file"  
	}  
}  
  
func shouldExtractContent(mimeType string) bool {  
	extractableTypes := []string{  
		"text/plain",  
		"text/html",  
		"application/pdf",  
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",  
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",  
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",  
	}  
	  
	for _, t := range extractableTypes {  
		if strings.Contains(mimeType, t) {  
			return true  
		}  
	}  
	return false  
}  
  
func extractTextFromBytes(content []byte, mimeType string) string {  
	// 简单的文本提取实现  
	// 在实际项目中，您可能需要使用专门的库来处理不同的文件格式  
	switch {  
	case strings.HasPrefix(mimeType, "text/"):  
		return string(content)  
	case strings.Contains(mimeType, "pdf"):  
		// 这里需要PDF解析库，如github.com/ledongthuc/pdf  
		return "PDF content extraction not implemented"  
	case strings.Contains(mimeType, "word"):  
		// 这里需要Word文档解析库  
		return "Word document content extraction not implemented"  
	default:  
		return ""  
	}  
}
7. 配置解析工具 (config.go)
package sharepoint  
  
import (  
	"fmt"  
	"time"  
  
	"infini.sh/coco/modules/common"  
	"infini.sh/framework/core/config"  
)  
  
func parseSharePointConfig(datasource *common.DataSource) (*SharePointConfig, error) {  
	if datasource.Connector.Config == nil {  
		return nil, fmt.Errorf("connector config is nil")  
	}  
	  
	cfg, err := config.NewConfigFrom(datasource.Connector.Config)  
	if err != nil {  
		return nil, fmt.Errorf("failed to parse config: %w", err)  
	}  
	  
	sharePointConfig := &SharePointConfig{}  
	err = cfg.Unpack(sharePointConfig)  
	if err != nil {  
		return nil, fmt.Errorf("failed to unpack config: %w", err)  
	}  
	  
	// 设置默认的重试配置  
	if sharePointConfig.RetryConfig.MaxRetries == 0 {  
		sharePointConfig.RetryConfig.MaxRetries = 3  
	}  
	if sharePointConfig.RetryConfig.InitialDelay == 0 {  
		sharePointConfig.RetryConfig.InitialDelay = time.Second  
	}  
	if sharePointConfig.RetryConfig.MaxDelay == 0 {  
		sharePointConfig.RetryConfig.MaxDelay = time.Minute  
	}  
	if sharePointConfig.RetryConfig.BackoffFactor == 0 {  
		sharePointConfig.RetryConfig.BackoffFactor = 2.0  
	}  
	  
	return sharePointConfig, nil  
}  
  
func validateSharePointConfig(config *SharePointConfig) error {  
	if config.SiteURL == "" {  
		return fmt.Errorf("site_url is required")  
	}  
	if config.TenantID == "" {  
		return fmt.Errorf("tenant_id is required")  
	}  
	if config.ClientID == "" {  
		return fmt.Errorf("client_id is required")  
	}  
	if config.ClientSecret == "" {  
		return fmt.Errorf("client_secret is required")  
	}  
	  
	return nil  
}