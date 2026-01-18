/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/connector"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

// s3Config defines S3 configuration.
//
// This is defined locally to avoid circular import with
// plugins/connectors/s3.
type s3Config struct {
	Endpoint        string `config:"endpoint"`
	AccessKeyID     string `config:"access_key_id"`
	SecretAccessKey string `config:"secret_access_key"`
	Bucket          string `config:"bucket"`
	UseSSL          bool   `config:"use_ssl"`
}

// getStringFromMap safely extracts a string from a map
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getBoolFromMap safely extracts a bool from a map
func getBoolFromMap(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func (h *APIHandler) createDoc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var obj = &core.Document{}
	err := h.DecodeJSON(req, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh
	err = orm.Create(ctx, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteCreatedOKJSON(w, obj.ID)
}

func (h *APIHandler) getDoc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("doc_id")

	obj := core.Document{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "document")
	exists, err := orm.GetV2(ctx, &obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":   id,
			"found": false,
		}, http.StatusNotFound)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"found":   true,
		"_id":     id,
		"_source": obj,
	}, 200)
}

func (h *APIHandler) getDocRawContent(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("doc_id")

	obj := core.Document{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "document")
	exists, err := orm.GetV2(ctx, &obj)
	if err != nil {
		h.WriteError(w, fmt.Sprintf("failed to acquire the document: %v", err), http.StatusInternalServerError)
		return
	}

	if !exists {
		h.WriteJSON(w, util.MapStr{
			"_id":   id,
			"found": false,
		}, http.StatusNotFound)
		return
	}

	// Handle empty URL
	if obj.URL == "" {
		h.WriteError(w, "document has no URL", http.StatusBadRequest)
		return
	}

	// Check if URL is raw content or external URL
	if obj.Metadata["url_is_raw_content"] == true {
		datasourceID := obj.Source.ID
		datasource, err := common.GetDatasourceConfig(ctx, datasourceID)
		if err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		connectorID := datasource.Connector.ConnectorID

		switch connectorID {
		case "s3":
			// Extract S3 configuration
			connectorConfig, ok := datasource.Connector.Config.(map[string]interface{})
			if !ok {
				h.WriteError(w, "failed to parse S3 config: invalid config type", http.StatusInternalServerError)
				return
			}
			cfg := s3Config{
				Endpoint:        getStringFromMap(connectorConfig, "endpoint"),
				AccessKeyID:     getStringFromMap(connectorConfig, "access_key_id"),
				SecretAccessKey: getStringFromMap(connectorConfig, "secret_access_key"),
				Bucket:          getStringFromMap(connectorConfig, "bucket"),
				UseSSL:          getBoolFromMap(connectorConfig, "use_ssl"),
			}

			if cfg.Endpoint == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" || cfg.Bucket == "" {
				h.WriteError(w, "s3 config is invalid: missing required fields", http.StatusInternalServerError)
				return
			}

			// Create minio client
			client, err := minio.New(cfg.Endpoint, &minio.Options{
				Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
				Secure: cfg.UseSSL,
			})
			if err != nil {
				h.WriteError(w, fmt.Sprintf("failed to create S3 client: %v", err), http.StatusInternalServerError)
				return
			}

			// Extract objectKey from document URL using net/url
			// URL format: http(s)://endpoint/bucket/objectKey
			u, err := url.Parse(obj.URL)
			if err != nil {
				h.WriteError(w, fmt.Sprintf("invalid S3 URL format: %s", obj.URL), http.StatusBadRequest)
				return
			}
			// u.Path will be like "/bucket/objectKey", trim the leading "/bucket/"
			prefix := "/" + cfg.Bucket + "/"
			if !strings.HasPrefix(u.Path, prefix) {
				h.WriteError(w, fmt.Sprintf("S3 URL path does not match bucket: %s", obj.URL), http.StatusBadRequest)
				return
			}
			objectKey := strings.TrimPrefix(u.Path, prefix)

			// Get object stream (does not download content yet)
			objStream, err := client.GetObject(req.Context(), cfg.Bucket, objectKey, minio.GetObjectOptions{})
			if err != nil {
				h.WriteError(w, fmt.Sprintf("failed to get S3 object: %v", err), http.StatusInternalServerError)
				return
			}
			defer objStream.Close()

			// Get object metadata
			info, err := objStream.Stat()
			if err != nil {
				// Check if it's a 404
				if minio.ToErrorResponse(err).Code == "NoSuchKey" {
					h.WriteJSON(w, util.MapStr{
						"_id":   id,
						"found": false,
					}, http.StatusNotFound)
					return
				}
				h.WriteError(w, fmt.Sprintf("failed to stat S3 object: %v", err), http.StatusInternalServerError)
				return
			}

			// Set HTTP headers
			contentType := info.ContentType
			if contentType == "" {
				// Fall back to detecting Content-Type from object key extension
				contentType = mime.TypeByExtension(filepath.Ext(objectKey))
				if contentType == "" {
					contentType = "application/octet-stream"
				}
			}
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))

			// Set Content-Disposition to inline for these file types
			// so that browser can preview them
			disposition := "attachment"
			if strings.HasPrefix(contentType, "image/") ||
				strings.HasPrefix(contentType, "video/") ||
				strings.HasPrefix(contentType, "audio/") ||
				contentType == "application/pdf" {
				disposition = "inline"
			}
			fileName := filepath.Base(objectKey)
			w.Header().Set("Content-Disposition", disposition+"; filename=\""+fileName+"\"")

			// Stream data directly from S3 to HTTP response
			_, err = io.Copy(w, objStream)
			if err != nil {
				log.Errorf("error streaming S3 object: %v", err)
			}
		case "local_fs":
			// Stream from local filesystem
			fileLocalPath := obj.URL

			// Open file
			file, err := os.Open(fileLocalPath)
			if err != nil {
				if os.IsNotExist(err) {
					h.WriteJSON(w, util.MapStr{
						"_id":   id,
						"found": false,
					}, http.StatusNotFound)
				} else {
					h.WriteError(w, fmt.Sprintf("failed to open file: %v", err), http.StatusInternalServerError)
				}
				return
			}
			defer file.Close()

			// Get file info
			fileInfo, err := file.Stat()
			if err != nil {
				h.WriteError(w, fmt.Sprintf("failed to stat file: %v", err), http.StatusInternalServerError)
				return
			}

			// Detect Content-Type
			contentType := mime.TypeByExtension(filepath.Ext(fileLocalPath))
			if contentType == "" {
				contentType = "application/octet-stream"
			}

			// Set HTTP headers
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

			// Set Content-Disposition to inline for these file types
			// so that browser can preview them
			disposition := "attachment"
			if strings.HasPrefix(contentType, "image/") ||
				strings.HasPrefix(contentType, "video/") ||
				strings.HasPrefix(contentType, "audio/") ||
				contentType == "application/pdf" {
				disposition = "inline"
			}
			fileName := filepath.Base(fileLocalPath)
			w.Header().Set("Content-Disposition", disposition+"; filename=\""+fileName+"\"")

			// Stream file content using http.ServeContent (handles range requests, etc.)
			http.ServeContent(w, req, filepath.Base(fileLocalPath), fileInfo.ModTime(), file)
		default:
			h.WriteError(w, fmt.Sprintf("unsupported connector: %s", connectorID), http.StatusBadRequest)
		}

	} else {
		// Redirect to external URL
		http.Redirect(w, req, obj.URL, http.StatusFound)
	}
}

func (h *APIHandler) updateDoc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("doc_id")
	ctx := orm.NewContextWithParent(req.Context())

	obj := core.Document{}
	err := h.DecodeJSON(req, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//protect
	obj.ID = id
	ctx.Refresh = orm.WaitForRefresh
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "document")

	//update share context
	ctx.Set(orm.SharingCheckingResourceCategoryEnabled, true)
	ctx.Set(orm.SharingResourceCategoryType, "datasource")
	ctx.Set(orm.SharingResourceCategoryFilterField, "source.id")
	ctx.Set(orm.SharingResourceCategoryID, obj.Source.ID)
	ctx.Set(orm.SharingResourceParentPath, obj.Category)
	ctx.Set(orm.SharingCheckingInheritedRulesEnabled, true)

	err = orm.Save(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteUpdatedOKJSON(w, obj.ID)
}

func (h *APIHandler) deleteDoc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("doc_id")

	obj := core.Document{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	ctx.Refresh = orm.WaitForRefresh
	err := orm.Delete(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteDeletedOKJSON(w, obj.ID)
}

func (h *APIHandler) searchDocs(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//handle url query args, convert to query builder
	builder, err := orm.NewQueryBuilderFromRequest(req, "title", "summary", "combined_fulltext")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Omit these fields. The frontend does not need them, and they are large enough
	// to slow us down.
	builder.Exclude("payload.*", "document_chunk", "ai_insights.embedding")
	builder.EnableBodyBytes()
	if len(builder.Sorts()) == 0 {
		builder.SortBy(orm.Sort{Field: "created", SortType: orm.DESC})
	}

	ctx := orm.NewContextWithParent(req.Context())
	view := h.GetParameter(req, "view")
	//view := "list"
	sourceIDs := builder.GetFilterValues("source.id")

	pathHierarchy := false
	//apply datasource filter //TODO datasource may support multi ids
	if len(sourceIDs) == 1 {
		ctx1 := orm.NewContext()
		ctx1.DirectReadAccess()
		ctx1.PermissionScope(security.PermissionScopePlatform)

		sourceIDArray, ok := sourceIDs[0].([]interface{})
		if ok {
			sourceID, ok := sourceIDArray[0].(string)
			if ok {
				ds, err := common.GetDatasourceConfig(ctx1, sourceID)
				if err != nil {
					panic(err)
				}
				if ds != nil {
					conn, err := connector.GetConnectorByID(ds.Connector.ConnectorID)
					if err != nil {
						panic(err)
					}
					if conn.PathHierarchy {
						pathHierarchy = true
					}

					ctx.Set(orm.SharingCheckingResourceCategoryEnabled, true)
					ctx.Set(orm.SharingResourceCategoryType, "datasource")
					ctx.Set(orm.SharingResourceCategoryFilterField, "source.id")
					ctx.Set(orm.SharingResourceCategoryID, ds.ID)
				}
			}
		}
	}

	//TODO cache
	var pathStr = "/"
	pathFilterStr := h.GetParameter(req, "path")
	if pathFilterStr != "" {
		array := []string{}
		err = util.FromJson(pathFilterStr, &array)
		if err != nil {
			panic(err)
		}
		if len(array) > 0 {
			pathStr = common.GetFullPathForCategories(array)
		}
	}

	//path str
	if view != "list" && pathHierarchy && pathStr != "" {
		builder.Filter(orm.TermQuery("_system.parent_path", pathStr))
		log.Trace("adding path hierarchy filter: ", pathStr)
		ctx.Set(orm.SharingResourceParentPath, pathStr)
	} else {
		//apply path filter to list view too
		if pathStr != "/" {
			builder.Filter(orm.TermQuery("_system.parent_path", pathStr))
			log.Trace("adding path hierarchy filter: ", pathStr)
			ctx.Set(orm.SharingResourceParentPath, pathStr)
		}
	}

	orm.WithModel(ctx, &core.Document{})
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "document")
	ctx.Set(orm.SharingCheckingInheritedRulesEnabled, true)

	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := elastic.SearchResponseWithMeta[core.Document]{}
	util.MustFromJSONBytes(res.Payload.([]byte), &result)

	nDocs := len(result.Hits.Hits)
	if nDocs > 0 {
		for i := range result.Hits.Hits {
			RefineIcon(req.Context(), &result.Hits.Hits[i].Source)
			RefineCoverThumbnail(req.Context(), &result.Hits.Hits[i].Source)
			RefineURL(req.Context(), &result.Hits.Hits[i].Source)
		}
	}

	h.WriteJSON(w, result, 200)
}

func (h *APIHandler) batchDeleteDoc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var ids []string
	err := h.DecodeJSON(req, &ids)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(ids) == 0 {
		h.WriteError(w, "document ids can not be empty", http.StatusBadRequest)
		return
	}

	builder, err := orm.NewQueryBuilderFromRequest(req, "title", "summary", "combined_fulltext")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	builder.Filter(orm.TermsQuery("id", ids))

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.Document{})

	_, err = orm.DeleteByQuery(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteAckOKJSON(w)
}
