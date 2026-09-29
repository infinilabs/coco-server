/* Common-task pipeline templates for the studio's new-pipeline gallery.
 * Every template only references processors from the live registry (the
 * gallery hides any template whose processors are not all installed), and
 * config values mirror the working enrich_documents reference chain —
 * processors self-gate by content type, so the chains stay unconditioned
 * like the shipped ones. */

export interface PipelineTemplate {
  id: string;
  icon: 'file-text' | 'file' | 'image' | 'video' | 'paperclip';
  suggestedName: string;
  processor: Record<string, any>[];
}

const TIKA = {
  tika_endpoint: 'http://127.0.0.1:9998',
  tika_timeout_in_seconds: 360
};

export const PIPELINE_TEMPLATES: PipelineTemplate[] = [
  {
    id: 'general-enrich',
    icon: 'file-text',
    suggestedName: 'enrich-documents',
    // the full knowledge-processing chain: parse, chunk, enrich, entity
    // extraction, vectorize — the document lands in the store with both the
    // inverted-index text fields and the knn vector fields (dual-write), so
    // every search engine can recall it
    processor: [
      { file_type_detection: {} },
      { file_metadata: {} },
      { generate_document_cover: {} },
      { document_text_attachment_extraction: { chunk_size: 7000, extract_attachments: true, ...TIKA } },
      { document_summarization: { ai_insights_max_length: 500, model_context_length: 128000 } },
      { extract_tags: { model_context_length: 128000 } },
      { extract_entities: { model_context_length: 128000, max_entities: 20 } },
      { document_embedding: {} }
    ]
  },
  {
    id: 'pdf',
    icon: 'file',
    suggestedName: 'pdf-processing',
    processor: [
      { file_type_detection: {} },
      { document_text_attachment_extraction: { chunk_size: 7000, extract_attachments: true, ...TIKA } },
      { document_summarization: { ai_insights_max_length: 500, model_context_length: 128000 } },
      { extract_tags: { model_context_length: 128000 } },
      { document_embedding: {} }
    ]
  },
  {
    id: 'word',
    icon: 'file',
    suggestedName: 'word-processing',
    processor: [
      { file_type_detection: {} },
      { document_text_attachment_extraction: { chunk_size: 7000, extract_attachments: true, ...TIKA } },
      { document_summarization: { ai_insights_max_length: 500, model_context_length: 128000 } },
      { extract_tags: { model_context_length: 128000 } }
    ]
  },
  {
    id: 'image',
    icon: 'image',
    suggestedName: 'image-processing',
    processor: [
      { file_type_detection: {} },
      { file_metadata: {} },
      { generate_document_cover: {} },
      // image files are described by the vision model inside text extraction
      // (configure vision_model_provider/vision_model when a vision model is
      // available); embedded image OCR also runs through Tika
      { document_text_attachment_extraction: { chunk_size: 7000, extract_attachments: true, ...TIKA } },
      { face_extraction: { pigo_facefinder_path: './config/ai/facefinder', ...TIKA } },
      { extract_tags: { model_context_length: 128000 } }
    ]
  },
  {
    id: 'video',
    icon: 'video',
    suggestedName: 'video-processing',
    processor: [
      { file_type_detection: {} },
      { file_metadata: {} },
      { document_summarization: { ai_insights_max_length: 500, model_context_length: 128000 } },
      { extract_tags: { model_context_length: 128000 } }
    ]
  },
  {
    id: 'attachment',
    icon: 'paperclip',
    suggestedName: 'attachment-processing',
    // the attachment side of the 8+2 story: extract the text, then give the
    // attachment a cover — two nodes, runs as the attachment pipeline
    processor: [
      { attachment_text_extraction: { ...TIKA } },
      { generate_attachment_cover: {} }
    ]
  }
];

// A template is offered only when every processor it names is installed.
export function availableTemplates(installedNames: Set<string>): PipelineTemplate[] {
  return PIPELINE_TEMPLATES.filter(t =>
    t.processor.every(entry => {
      const name = Object.keys(entry)[0];
      return installedNames.has(name);
    })
  );
}

export function templateEntryNames(t: PipelineTemplate): string[] {
  return t.processor.map(entry => Object.keys(entry)[0]);
}
