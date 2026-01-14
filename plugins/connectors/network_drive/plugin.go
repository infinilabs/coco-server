/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package network_drive

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"path/filepath"
	"strings"
	"time"

	"infini.sh/coco/core"

	log "github.com/cihub/seelog"
	"github.com/hirochachacha/go-smb2"
	"infini.sh/coco/plugins/connectors"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
)

const (
	ConnectorNetworkDrive = "network_drive"
	ConnectionTimeout     = time.Duration(5) * time.Second
)

// Config defines the configuration for the network drive connector.
// It supports credentials-based scanning (for direct SMB connections).
type Config struct {
	//  Credentials-based Auth (SMB) Options
	Endpoint   string   `config:"endpoint"`
	Share      string   `config:"share"`
	Username   string   `config:"username"`
	Password   string   `config:"password"`
	Domain     string   `config:"domain"` // Optional, e.g., "WORKGROUP"
	Paths      []string `config:"paths"`
	Extensions []string `config:"extensions"`
}

type Plugin struct {
	cmn.ConnectorProcessorBase
}

func init() {
	pipeline.RegisterProcessorPlugin(ConnectorNetworkDrive, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	runner := Plugin{}
	runner.Init(c, &runner)
	return &runner, nil
}

func (p *Plugin) Name() string {
	return ConnectorNetworkDrive
}

func (p *Plugin) Fetch(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {
	cfg := Config{}
	p.MustParseConfig(datasource, &cfg)

	log.Debugf("[%s connector] handling datasource: %v", ConnectorNetworkDrive, cfg)

	if cfg.Endpoint == "" || cfg.Share == "" || cfg.Username == "" {
		return fmt.Errorf("missing required fields for credentials-based auth for data source [%s]: endpoint, share, or username", datasource.Name)
	}

	conn, err := net.DialTimeout("tcp", cfg.Endpoint, ConnectionTimeout)
	if err != nil {
		return fmt.Errorf("failed to dial SMB server %s for data source: [%s]: %v", cfg.Endpoint, datasource.Name, err)
	}
	defer func() {
		_ = conn.Close()
	}()

	var dialer = &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     cfg.Username,
			Password: cfg.Password,
			Domain:   cfg.Domain,
		},
	}
	scanCtx := ctx
	deadline := time.Now().Add(ConnectionTimeout)
	dialCtx, cancelOnTimeout := context.WithDeadline(scanCtx, deadline)
	defer cancelOnTimeout() // release resource even though not time out

	session, err := dialer.DialContext(dialCtx, conn)
	if err != nil {
		return fmt.Errorf("failed to dial SMB server %s for data source: [%s]: %v", cfg.Endpoint, datasource.Name, err)
	}
	defer func() {
		_ = session.Logoff()
	}()

	share, err := session.Mount(cfg.Share)
	if err != nil {
		return fmt.Errorf("failed to mount SMB share '%s' on server %s for datasource [%s]: %v", cfg.Share, cfg.Endpoint, datasource.Name, err)
	}
	defer func() {
		_ = share.Umount()
	}()

	log.Debugf("[%s connector] connecting to SMB share: //%s/%s for data source: %s", ConnectorNetworkDrive, cfg.Endpoint, cfg.Share, datasource.Name)

	// A map for extensions
	extMap := make(map[string]bool)
	for _, ext := range cfg.Extensions {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extMap[strings.ToLower(ext)] = true
	}

	// Track all unique folder paths that contain matching files
	foldersWithMatchingFiles := make(map[string]bool)

	for _, path := range cfg.Paths {
		err = fs.WalkDir(share.DirFS("."), path, func(currentPath string, d fs.DirEntry, err error) error {
			if global.ShuttingDown() {
				return errors.New("system is shutting down, scan cancelled")
			}

			if err != nil {
				_ = log.Warnf("[%s connector] error accessing SMB path %q: %v", ConnectorNetworkDrive, currentPath, err)
				return err
			}

			if d.IsDir() {
				return nil
			}

			// Check file extension name
			fileExt := strings.ToLower(filepath.Ext(currentPath))

			// Extension name not matched
			if len(extMap) > 0 && !extMap[fileExt] {
				return nil
			}

			// Mark all parent folders as containing matching files
			connectors.MarkParentFoldersAsValid(filepath.ToSlash(currentPath), foldersWithMatchingFiles)

			p.processFile(ctx, d, filepath.ToSlash(currentPath), &cfg, connector, datasource)
			return nil
		})

		if err != nil {
			_ = log.Errorf("[%s connector] error walking SMB share '%s' for datasource [%s]: %v", ConnectorNetworkDrive, cfg.Share, datasource.Name, err)
		}
	}

	// Now create folder documents for all folders that contain matching files
	p.createFolderDocuments(ctx, foldersWithMatchingFiles, connector, datasource, &cfg)

	log.Infof("[%s connector] finished fetching datasource [%s]", ConnectorNetworkDrive, datasource.Name)
	return nil
}

// processFile is a helper function to filter, transform, and queue a single file.
func (p *Plugin) processFile(ctx *pipeline.Context, d fs.DirEntry, currentPath string, cfg *Config, connector *core.Connector, datasource *core.DataSource) {

	// Construct a full UNC-style path for the URL field
	fullPath := fmt.Sprintf("//%s/%s/%s", cfg.Endpoint, cfg.Share, currentPath)

	fileInfo, err := d.Info()
	if err != nil {
		_ = log.Warnf("[%s connector] failed to get file info for %q: %v", ConnectorNetworkDrive, fullPath, err)
		return
	}

	// Create file document using helper
	parentCategoryArray := connectors.BuildParentCategoryArray(currentPath)
	title := fileInfo.Name()
	idSuffix := fmt.Sprintf("%s-%s-%s", cfg.Endpoint, cfg.Share, currentPath)

	doc := connectors.CreateDocumentWithHierarchy(connectors.TypeFile, connectors.TypeFile, title, fullPath, int(fileInfo.Size()),
		parentCategoryArray, datasource, idSuffix)

	modTime := fileInfo.ModTime()
	doc.Created = &modTime
	doc.Updated = &modTime

	p.Collect(ctx, connector, datasource, doc)
}

// createFolderDocuments creates document entries for all folders that contain matching files
func (p *Plugin) createFolderDocuments(ctx *pipeline.Context, foldersWithMatchingFiles map[string]bool, connector *core.Connector, datasource *core.DataSource, cfg *Config) {
	var docs []core.Document
	for folderPath := range foldersWithMatchingFiles {
		if global.ShuttingDown() {
			log.Info("[network_drive connector] Shutdown signal received, stopping folder creation.")
			break
		}
		folderName := filepath.Base(folderPath)
		parentCategoryArray := connectors.BuildParentCategoryArray(folderPath)
		url := fmt.Sprintf("//%s/%s/%s/", cfg.Endpoint, cfg.Share, folderPath)
		idSuffix := fmt.Sprintf("%s-%s-folder-%s", cfg.Endpoint, cfg.Share, folderPath)

		doc := connectors.CreateDocumentWithHierarchy(connectors.TypeFolder, connectors.IconFolder, folderName, url, 0,
			parentCategoryArray, datasource, idSuffix)

		docs = append(docs, doc)
	}

	if len(docs) > 0 {
		p.BatchCollect(ctx, connector, datasource, docs)
	}
}
