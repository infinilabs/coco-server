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
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/hirochachacha/go-smb2"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
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
	connectors.BasePlugin
	// mu protects the cancel function below.
	mu sync.Mutex
	// ctx is the root context for the plugin, created on Start and cancelled on Stop.
	ctx context.Context
	// cancel is the function to call to cancel a running scan.
	cancel context.CancelFunc
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}

func (p *Plugin) Setup() {
	p.BasePlugin.Init("connector.network_drive", "indexing network drive", p)
}

func (p *Plugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ctx, p.cancel = context.WithCancel(context.Background())
	return p.BasePlugin.Start(connectors.DefaultSyncInterval)
}

func (p *Plugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		log.Infof("[%v connector] received stop signal, cancelling current scan", ConnectorNetworkDrive)
		p.cancel()
		p.ctx = nil
		p.cancel = nil
	}
	return nil
}

func (p *Plugin) Name() string {
	return ConnectorNetworkDrive
}

// Scan is the main entry point that dispatches to the correct scanning method.
func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	cfg := Config{}
	if err := connectors.ParseConnectorConfigure(connector, datasource, &cfg); err != nil {
		_ = log.Errorf("[%v connector] parsing connector configuration failed for datasource [%s]: %v", ConnectorNetworkDrive, datasource.Name, err)
		return
	}
	p.scanSmbShare(datasource, &cfg)
}

// scanSmbShare handles scanning a remote SMB share using provided credentials.
func (p *Plugin) scanSmbShare(datasource *common.DataSource, cfg *Config) {
	// Get the parent context
	p.mu.Lock()
	parentCtx := p.ctx
	p.mu.Unlock()

	// Check if the plugin has been stopped before proceeding.
	if parentCtx == nil {
		_ = log.Warnf("[%v connector] plugin is stopped, skipping scan for datasource [%s]", ConnectorNetworkDrive, datasource.Name)
		return
	}

	if cfg.Endpoint == "" || cfg.Share == "" || cfg.Username == "" {
		_ = log.Errorf("[%v connector] missing required fields for credentials-based auth for data source [%s]: endpoint, share, or username", ConnectorNetworkDrive, datasource.Name)
		return
	}

	conn, err := net.DialTimeout("tcp", cfg.Endpoint, ConnectionTimeout)
	if err != nil {
		_ = log.Errorf("[%v connector] failed to dial SMB server %s for data source: [%s]: %v", ConnectorNetworkDrive, cfg.Endpoint, datasource.Name, err)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	dialer := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     cfg.Username,
			Password: cfg.Password,
			Domain:   cfg.Domain,
		},
	}

	deadline := time.Now().Add(ConnectionTimeout)
	dialCtx, cancelOnTimeout := context.WithDeadline(parentCtx, deadline)
	defer cancelOnTimeout() // release resource even though not time out

	session, err := dialer.DialContext(dialCtx, conn)
	if err != nil {
		_ = log.Errorf("[%v connector] failed to dial SMB server %s for data source: [%s]: %v", ConnectorNetworkDrive, cfg.Endpoint, datasource.Name, err)
		return
	}
	defer func() {
		_ = session.Logoff()
	}()

	share, err := session.Mount(cfg.Share)
	if err != nil {
		_ = log.Errorf("[%v connector] failed to mount SMB share '%s' on server %s for datasource [%s]: %v", ConnectorNetworkDrive, cfg.Share, cfg.Endpoint, datasource.Name, err)
		return
	}
	defer func() {
		_ = share.Umount()
	}()

	log.Debugf("[%v connector] connecting to SMB share: //%s/%s for data source: %s", ConnectorNetworkDrive, cfg.Endpoint, cfg.Share, datasource.Name)

	scanCtx, scanCancel := context.WithCancel(parentCtx)
	defer scanCancel()

	// A map for extensions
	extMap := make(map[string]bool)
	for _, ext := range cfg.Extensions {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extMap[strings.ToLower(ext)] = true
	}

	for _, path := range cfg.Paths {
		err = fs.WalkDir(share.DirFS("."), path, func(currentPath string, d fs.DirEntry, err error) error {
			select {
			case <-scanCtx.Done():
				return errors.New("network drive connector scan cancelled")
			default:
			}

			if global.ShuttingDown() {
				return errors.New("system is shutting down, scan cancelled")
			}

			if err != nil {
				_ = log.Warnf("[%v connector] error accessing SMB path %q: %v", ConnectorNetworkDrive, currentPath, err)
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

			p.processFile(d, filepath.ToSlash(currentPath), cfg, datasource)
			return nil
		})

		if err != nil {
			_ = log.Errorf("[%v connector] error walking SMB share '%s' for datasource [%s]: %v", ConnectorNetworkDrive, cfg.Share, datasource.Name, err)
		}
	}
}

// processFile is a helper function to filter, transform, and queue a single file.
func (p *Plugin) processFile(d fs.DirEntry, currentPath string, cfg *Config, datasource *common.DataSource) {

	// Construct a full UNC-style path for the URL field
	fullPath := fmt.Sprintf("//%s/%s/%s", cfg.Endpoint, cfg.Share, currentPath)

	fileInfo, err := d.Info()
	if err != nil {
		_ = log.Warnf("[%v connector] failed to get file info for %q: %v", ConnectorNetworkDrive, fullPath, err)
		return
	}

	modTime := fileInfo.ModTime()
	doc := common.Document{
		Source:   common.DataSourceReference{ID: datasource.ID, Type: "connector", Name: datasource.Name},
		Type:     ConnectorNetworkDrive,
		Icon:     "default",
		Title:    fileInfo.Name(),
		Category: filepath.Dir(fmt.Sprintf("%s/%s", cfg.Share, currentPath)),
		Content:  "", // skip content
		URL:      fullPath,
		Size:     int(fileInfo.Size()),
	}
	doc.Created = &modTime
	doc.Updated = &modTime
	doc.ID = util.MD5digest(fmt.Sprintf("%s-%s", datasource.ID, fullPath))

	data := util.MustToJSONBytes(doc)
	if err := queue.Push(p.Queue, data); err != nil {
		_ = log.Errorf("[%v connector] failed to push document to queue for data source [%s]: %v", ConnectorNetworkDrive, datasource.Name, err)
	}
}
