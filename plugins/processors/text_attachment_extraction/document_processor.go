/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package text_attachment_extraction

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/common/fingerprint"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/coco/plugins/connectors/local_fs"
	"infini.sh/coco/plugins/connectors/s3"
	utils "infini.sh/coco/plugins/processors"
	"infini.sh/coco/plugins/processors/fileproc"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const DocumentProcessorName = "document_text_attachment_extraction"

var supportedConnectors = map[string]bool{
	s3.ConnectorS3:            true,
	local_fs.ConnectorLocalFs: true,
}

func init() {
	pipeline.RegisterProcessorPlugin(DocumentProcessorName, NewDocumentProcessor)
}

// DocumentTextAttachmentExtractionProcessor extracts text and attachments from document files.
type DocumentTextAttachmentExtractionProcessor struct {
	config      *DocumentConfig
	outputQueue *queue.QueueConfig
}

// DocumentConfig holds configuration for the document_text_attachment_extraction processor.
// TikaEndpoint / TikaTimeoutInSeconds are only used when processing file types
// that require Apache Tika (e.g. PDF, DOCX).  PPTX and plain images are
// handled without Tika.
type DocumentConfig struct {
	MessageField param.ParaKey      `config:"message_field"`
	OutputQueue  *queue.QueueConfig `config:"output_queue"`

	TikaEndpoint         string `config:"tika_endpoint"`
	TikaTimeoutInSeconds int    `config:"tika_timeout_in_seconds"`
	ChunkSize            int    `config:"chunk_size"`

	// ExtractAttachments controls whether embedded attachments (images, etc.)
	// are extracted from documents. When false, attachment extraction, OCR,
	// image markers, and upload are all skipped. Defaults to true.
	ExtractAttachments *bool `config:"extract_attachments"`

	// Remote parse backend (W2 L1): when set, listed extensions parse
	// through this service FIRST (first-parser semantics — a non-empty
	// answer wins, any failure falls back to the built-in Tika route).
	// Per-datasource pipelines pick different backends by configuring
	// their own copy of this processor.
	RemoteParseURL        string   `config:"remote_parse_url"`
	RemoteParseTimeoutSec int      `config:"remote_parse_timeout_in_seconds"`
	RemoteParseExts       []string `config:"remote_parse_exts"`

	// XlsxFirstRowAsHeader treats the first row of each sheet as column
	// names for the `列名: 值` row records (W2 structured Excel route).
	// Defaults to true — header-bearing business sheets are the common
	// ingestion case; turn it off for headerless exports.
	XlsxFirstRowAsHeader *bool `config:"xlsx_first_row_as_header"`

	// Vision model used for image-file description
	VisionModelProviderID string `config:"vision_model_provider"`
	VisionModelName       string `config:"vision_model"`
	ImageContentFormat    string `config:"image_content_format"`

	// BCP 47 language tag for LLM-generated content (e.g. "en-US", "zh-CN")
	LLMGenerationLang string `config:"llm_generation_lang"`
}

func NewDocumentProcessor(c *config.Config) (pipeline.Processor, error) {
	cfg := DocumentConfig{
		MessageField:       core.PipelineContextDocuments,
		ImageContentFormat: "data_uri",
	}
	if err := c.Unpack(&cfg); err != nil {
		return nil, err
	}

	if cfg.LLMGenerationLang == "" {
		if appCfg := common.AppConfig(); appCfg.DocumentProcessing != nil && appCfg.DocumentProcessing.LLMGenerationLanguage != "" {
			cfg.LLMGenerationLang = appCfg.DocumentProcessing.LLMGenerationLanguage
		}
	}
	cfg.LLMGenerationLang = utils.ValidateAndNormalizeLLMLang(DocumentProcessorName, cfg.LLMGenerationLang)

	if cfg.ChunkSize <= 0 {
		panic(fmt.Sprintf("processor [%s] configuration [chunk_size] is not set or invalid, should be a positive number", DocumentProcessorName))
	}

	// Default ExtractAttachments to true if not explicitly set.
	if cfg.ExtractAttachments == nil {
		defaultTrue := true
		cfg.ExtractAttachments = &defaultTrue
	}

	// Default XlsxFirstRowAsHeader to true if not explicitly set.
	if cfg.XlsxFirstRowAsHeader == nil {
		defaultHeader := true
		cfg.XlsxFirstRowAsHeader = &defaultHeader
	}

	p := &DocumentTextAttachmentExtractionProcessor{config: &cfg}
	if cfg.OutputQueue != nil {
		p.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}
	return p, nil
}

func (p *DocumentTextAttachmentExtractionProcessor) Name() string {
	return DocumentProcessorName
}

func (p *DocumentTextAttachmentExtractionProcessor) Process(ctx *pipeline.Context) error {
	obj := ctx.Get(p.config.MessageField)
	if obj == nil {
		log.Warnf("processor [%s] receives an empty pipeline context", p.Name())
		return nil
	}

	messages, ok := obj.([]queue.Message)
	if !ok {
		return nil
	}

	enqueued := make(map[int]bool)

	for i := range messages {
		if global.ShuttingDown() {
			log.Debugf("[%s] shutting down, skipping remaining %d documents", p.Name(), len(messages)-i)
			return fmt.Errorf("shutting down")
		}

		doc := core.Document{}
		if err := util.FromJSONBytes(messages[i].Data, &doc); err != nil {
			log.Errorf("processor [%s] failed to deserialize document: %s", p.Name(), err)
			continue
		}

		connectorID, err := utils.GetConnectorID(&doc)
		if err != nil && doc.ID != "" {
			log.Debugf("processor [%s] no connector for document [%s]: %v", p.Name(), doc.ID, err)
		}

		if err != nil || !supportedConnectors[connectorID] || doc.Type != connectors.TypeFile {
			// Content-only documents (API-created, webhook-synced, edited
			// through the document API) never go through file extraction,
			// but they still deserve chunks so the embedding stage has
			// something to vectorize — without this branch a POST /document
			// document stays chunk-less forever.
			if p.chunkContentOnly(&doc) {
				messages[i].Data = util.MustToJSONBytes(doc)
			}
			continue
		}

		log.Infof("processor [%s] processing file [%s/%s] from connector [%s]", p.Name(), doc.Title, doc.ID, connectorID)
		if err := p.processDocument(ctx.Context, &doc, connectorID); err != nil {
			log.Errorf("processor [%s] failed to process [%s/%s]: %s", p.Name(), doc.Title, doc.ID, err)
			continue
		}

		messages[i].Data = util.MustToJSONBytes(doc)

		if p.outputQueue != nil {
			if err := queue.Push(p.outputQueue, messages[i].Data); err != nil {
				log.Errorf("processor [%s] failed to push document [%s/%s] to output queue: %v", p.Name(), doc.Title, doc.ID, err)
			} else {
				enqueued[i] = true
			}
		}
	}

	if p.outputQueue != nil {
		for i := range messages {
			if !enqueued[i] {
				if err := queue.Push(p.outputQueue, messages[i].Data); err != nil {
					log.Errorf("processor [%s] failed to push skipped document [%d] to output queue: %v", p.Name(), i, err)
				}
			}
		}
	}

	return nil
}

func (p *DocumentTextAttachmentExtractionProcessor) processDocument(ctx context.Context, doc *core.Document, connectorID string) error {
	tempDir, err := os.MkdirTemp("", "coco-text-extraction-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	log.Tracef("[%s] downloading file for [%s/%s]", p.Name(), doc.Title, doc.ID)
	localPath, err := fileproc.DownloadToLocal(ctx, doc, connectorID, tempDir)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	if global.ShuttingDown() {
		return fmt.Errorf("shutting down")
	}

	if err := p.extractTextAndAttachment(ctx, doc, localPath); err != nil {
		return err
	}

	log.Debugf("processor [%s] extracted text/attachments for [%s/%s]", p.Name(), doc.Title, doc.ID)
	return nil
}

// extractTextAndAttachment dispatches to the correct extractor based on file extension.
// chunkContentOnly gives non-file documents (API-created, webhook-synced,
// edited through the document API) chunks cut straight from their content,
// mirroring what file extraction produces for uploads. Always recomputes —
// an edit that changed content must not keep the previous chunks — and
// reports whether anything changed so the caller re-serializes only then.
func (p *DocumentTextAttachmentExtractionProcessor) chunkContentOnly(doc *core.Document) bool {
	if doc == nil || strings.TrimSpace(doc.Content) == "" {
		return false
	}
	doc.Chunks = fileproc.SplitStructuredPages([]string{doc.Content}, fileproc.StructuredChunkConfig{
		ChunkSize: p.config.ChunkSize,
	})
	return true
}

func (p *DocumentTextAttachmentExtractionProcessor) extractTextAndAttachment(ctx context.Context, doc *core.Document, localPath string) error {
	ext := strings.ToLower(filepath.Ext(localPath))

	// Magic-number correction first (W2 L0): filenames lie, and only the
	// native zip/xml routes below care — Tika content-sniffs on its own.
	routedExt, mismatch := effectiveExt(localPath, ext)
	if mismatch != "" {
		log.Infof("processor [%s] document [%s]: %s (routing as %s)", p.Name(), doc.Title, mismatch, routedExt)
		if doc.Metadata == nil {
			doc.Metadata = map[string]interface{}{}
		}
		doc.Metadata[mimeMismatchTag] = mismatch
	}

	var (
		extraction fileproc.Extraction
		err        error
	)

	// W2 L1 first-parser chain: the remote backend runs FIRST for its
	// configured extensions and wins on a non-empty answer; any failure
	// falls through to the built-in routes below
	if pages, won := p.tryRemoteParseBackend(ctx, doc, routedExt, localPath); won {
		extraction.Pages = pages
	} else {

		switch routedExt {
		case ".pdf":
			extraction, err = p.processPdf(ctx, doc, localPath)
		case ".pptx", ".pptm":
			extraction, err = p.processPptx(ctx, doc, localPath)
		case ".xlsx":
			// W2 structured route: one page per sheet, `列名: 值` rows —
			// falls back to Tika when the native reader cannot handle it
			var xerr error
			extraction.Pages, xerr = parseXlsx(localPath, p.config.XlsxFirstRowAsHeader == nil || *p.config.XlsxFirstRowAsHeader)
			if xerr != nil {
				log.Warnf("processor [%s] native xlsx route failed for [%s] (%v), falling back to tika", p.Name(), doc.Title, xerr)
				extraction, err = p.processPdf(ctx, doc, localPath)
			}
		case ".epub":
			var eerr error
			extraction.Pages, eerr = parseEpub(localPath)
			if eerr != nil {
				log.Warnf("processor [%s] native epub route failed for [%s] (%v), falling back to tika", p.Name(), doc.Title, eerr)
				extraction, err = p.processPdf(ctx, doc, localPath)
			}
		case ".xmind":
			// no Tika fallback exists for XMind — the native parser is the backend
			extraction.Pages, err = parseXmind(localPath)
		case ".mhtml", ".mht":
			data, merr := os.ReadFile(localPath)
			if merr == nil {
				extraction.Pages, merr = parseMhtml(data)
			}
			if merr != nil {
				log.Warnf("processor [%s] native mhtml route failed for [%s] (%v), falling back to tika", p.Name(), doc.Title, merr)
				extraction, err = p.processPdf(ctx, doc, localPath)
			}
		case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff", ".tif":
			extraction, err = p.processImage(ctx, localPath)
			if err == nil {
				// image documents carry their perceptual hash in metadata
				// (W12) — near-duplicate image grouping keys on it; decode
				// failure leaves it unset, same "no fingerprint" convention
				if data, rerr := os.ReadFile(localPath); rerr == nil {
					if ph := fingerprint.ImagePhash(data); ph != 0 {
						if doc.Metadata == nil {
							doc.Metadata = map[string]interface{}{}
						}
						doc.Metadata["image_phash"] = int64(ph)
					}
				}
			}
		default:
			// OLE payloads (.doc/.xls/.ppt/.ppt-ole) and everything else go
			// through Tika, which sniffs the real format from the bytes
			extraction, err = p.processPdf(ctx, doc, localPath)
		}

	}

	if err != nil {
		return err
	}

	// Page-level cleanup (W2 L0): drop repeating header/footer furniture and
	// flag garbled pages — both conservative, both visible in metadata.
	extraction.Pages = stripRepeatingHeaderFooter(extraction.Pages)
	if garbled := detectGarbledPages(extraction.Pages); len(garbled) > 0 {
		log.Warnf("processor [%s] document [%s]: garbled pages detected %v (marked for the timeline; forced-OCR reroute is an L1 backend concern)", p.Name(), doc.Title, garbled)
		if doc.Metadata == nil {
			doc.Metadata = map[string]interface{}{}
		}
		doc.Metadata["garbled_pages"] = garbled
	}

	// W3/D10: structure-aware chunking — headings split sections, each
	// chunk carries its breadcrumb, hard cap + bounded overlap. Unmarked
	// input (legacy docs, plain text) lands in one root section and still
	// windows better than the old mid-word guillotine.
	doc.Chunks = fileproc.SplitStructuredPages(extraction.Pages, fileproc.StructuredChunkConfig{
		ChunkSize: p.config.ChunkSize,
	})
	doc.Attachments = extraction.Attachments
	doc.Content = strings.Join(extraction.Pages, " ")
	return nil
}
