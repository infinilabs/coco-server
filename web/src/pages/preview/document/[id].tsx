import { Button, Drawer, Result } from 'antd';
import { useParams } from 'react-router-dom';
import { highlightPhrase, highlightTerms } from './highlight';
import { ArrowLeft, Wrench } from 'lucide-react';

import { request } from '@/service/request';
import { fetchEntityUser } from '@/service/api/entity';
import loadingIcon from '@/assets/svg-icon/file-loading.svg';
import logoLight from '@/assets/imgs/coco-logo-text-light.svg';
import logoDark from '@/assets/imgs/coco-logo-text-dark.svg';
import { getDarkMode } from '@/store/slice/theme';
import { useAppSelector } from '@/hooks/business/useStore';
import classNames from 'classnames';

import PreviewContent from './components/PreviewContent';
import ProcessingPanel from './components/ProcessingPanel';
import { ensureFilenameExtension, extensionFromMime } from 'ui-search/source';

// Helper function to extract a filename from a Content-Disposition header.
// Supported forms include:
//   attachment; filename="example.pdf"
//   inline; filename="example.pdf"
function parseFilenameFromContentDisposition(header: string | null, fallback: string): string {
  if (!header) return fallback;
  const match = header.match(/filename="([^"]+)"/);
  if (match?.[1]) return match[1];
  return fallback;
}

export function Component() {
  const { id } = useParams();
  const [searchParams] = useSearchParams();
  const mode = searchParams.get('mode');
  const appIntegrationId = searchParams.get('app-integration-id');

  const embedded = mode === 'embedded';
  const darkMode = useAppSelector(getDarkMode);
  const theme = darkMode ? 'dark' : 'light';

  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<any>();
  const [error, setError] = useState<any>();
  const [redirectUrl, setRedirectUrl] = useState<string>();
  const [contentBlobUrl, setContentBlobUrl] = useState<string>();
  const [downloadFilename, setDownloadFilename] = useState<string>();
  const [rawContentError, setRawContentError] = useState<string>();
  const [processingOpen, setProcessingOpen] = useState(false);
  // W17 highlight: ?quote= is the cited chunk's exact excerpt (precise
  // landing, tier 2); ?q= carries the query terms (term marks, tier 3)
  const highlightQuery = searchParams.get('q') ?? '';
  const highlightQuote = searchParams.get('quote') ?? '';
  const contentRef = useRef<HTMLDivElement>(null);
  const { t } = useTranslation();
  const navigate = useNavigate();

  // Search results open this page with window.open, so the tab usually has no
  // history to go back to. Close the tab when an opener exists (returning the
  // user to the results they came from), use in-tab history when present, and
  // fall back to home for direct links.
  const handleBack = () => {
    if ((history.state?.idx ?? 0) > 0) {
      navigate(-1);
      return;
    }
    if (window.opener) {
      window.close();
      return;
    }
    navigate('/');
  };

  // requestHeaders is used by the Preview sub-components when fetching the
  // raw_content URL. We pass the app-integration-id header so embedded/widget
  // preview works consistently.
  const requestHeaders = useMemo(() => {
    return appIntegrationId ? { 'APP-INTEGRATION-ID': appIntegrationId } : undefined;
  }, [appIntegrationId]);

  // Inspect the raw_content endpoint to decide whether the document is an
  // external link or a file stream. The endpoint returns a JSON wrapper with the
  // external URL for redirects, and a file stream for raw content. We use the
  // X-Document-Redirect header to distinguish the two without relying on a 302
  // status, which is opaque and unreadable in cross-origin contexts.
  const inspectRawContent = async (docData: any) => {
    const rawContent: string | undefined = docData?.metadata?.raw_content;
    if (!rawContent) return;

    try {
      const res = await fetch(rawContent, {
        headers: requestHeaders
      });

      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
      }

      if (res.headers.get('X-Document-Redirect')) {
        const payload = await res.json();
        setRedirectUrl(payload.url);
        return;
      }

      const blob = await res.blob();
      // the Content-Disposition name (or the title fallback) often carries no
      // extension — a file saved that way can't be opened, so derive one from
      // the document metadata or the blob's own content type
      const { file_extension, mime_type, content_type } = docData?.metadata ?? {};
      const extension =
        file_extension?.replace(/^\./, '') ||
        extensionFromMime(mime_type) ||
        extensionFromMime(blob.type) ||
        '';
      const filename = ensureFilenameExtension(
        parseFilenameFromContentDisposition(res.headers.get('Content-Disposition'), docData.title || 'download'),
        extension
      );

      // for images, tag the blob URL with the filename as a fragment: the URL
      // keeps resolving, but a copied address / save-as now carries a real
      // name with an extension instead of the bare blob UUID
      const blobUrl =
        content_type === 'image'
          ? `${URL.createObjectURL(blob)}#${encodeURIComponent(filename)}`
          : URL.createObjectURL(blob);
      setContentBlobUrl(blobUrl);
      setDownloadFilename(filename);

      // Overwrite the raw_content URL with the Blob URL. The Preview
      // sub-components read metadata.raw_content and fetch from it; by giving
      // them a Blob URL we avoid a second network request and ensure the
      // rendered content is exactly what raw_content returned.
      setData((prev: any) => ({
        ...prev,
        metadata: {
          ...prev?.metadata,
          raw_content: blobUrl
        }
      }));
    } catch (err) {
      setRawContentError(err instanceof Error ? err.message : String(err));
    }
  };

  useAsyncEffect(async () => {
    try {
      const { data } = await request({
        method: 'get',
        url: `/document/${id}`,
        headers: requestHeaders
      });

      const dataSource = data._source;

      let ownerData;
      const ownerId = dataSource?._system?.owner_id;
      if (ownerId) {
        const { data } = await fetchEntityUser({ id: ownerId }, { headers: requestHeaders, ignoreError: true });
        ownerData = data;
      }

      const enrichedData = {
        ...dataSource,
        owner: ownerData
      };

      setData(enrichedData);

      // We must inspect the raw_content endpoint before rendering the preview,
      // because the endpoint returns either a file stream or a JSON wrapper with
      // the external URL. Replacing metadata.raw_content with a Blob URL is the
      // cleanest way to let the Preview components render the fetched content
      // without extra fetch logic.
      await inspectRawContent(enrichedData);
    } catch (error) {
      if (error instanceof Error) {
        setError(error.message);
      } else {
        setError(error);
      }
    } finally {
      setLoading(false);
    }
  }, [appIntegrationId, embedded, id, requestHeaders]);

  // Revoke the Blob URL on unmount to avoid leaking memory.
  useEffect(() => {
    return () => {
      if (contentBlobUrl) {
        URL.revokeObjectURL(contentBlobUrl);
      }
    };
  }, [contentBlobUrl]);

  // highlight after the content renders (blob load + paint settle).
  // quote landing takes priority: one precise passage beats scattered terms.
  useEffect(() => {
    if (loading || error || !contentRef.current) return;
    if (!highlightQuote && !highlightQuery) return;
    const timer = setTimeout(() => {
      let first: HTMLElement | undefined;
      if (highlightQuote) {
        first = highlightPhrase(contentRef.current, highlightQuote);
      }
      if (!first && highlightQuery) {
        const result = highlightTerms(contentRef.current, highlightQuery);
        first = result.firstElement;
      }
      first?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }, 400);
    return () => clearTimeout(timer);
  }, [loading, error, highlightQuery, highlightQuote, contentBlobUrl, data]);

  const renderContent = () => {
    if (loading) {
      return (
        <div className='fixed-center'>
          <img className='h-64px w-64px' src={loadingIcon} alt='' />
        </div>
      );
    }

    if (error) {
      return (
        <div className='h-full flex flex-col justify-center'>
          <Result
            status='404'
            subTitle={String(error)}
            title={t('page.preview.hints.failed')}
            extra={
              <Button
                type='primary'
                onClick={() => {
                  window.location.reload();
                }}
              >
                {t('page.preview.buttons.reload')}
              </Button>
            }
          />
        </div>
      );
    }

    // When raw_content inspection failed but we still have document metadata,
    // show a non-fatal warning so the user can retry.
    if (rawContentError && !contentBlobUrl && !redirectUrl) {
      return (
        <div className='h-full flex flex-col justify-center'>
          <Result
            status='warning'
            subTitle={rawContentError}
            title={t('page.preview.hints.failed')}
            extra={
              <Button
                type='primary'
                onClick={() => {
                  setRawContentError(undefined);
                  inspectRawContent(data);
                }}
              >
                {t('page.preview.buttons.reload')}
              </Button>
            }
          />
        </div>
      );
    }

    return (
      <PreviewContent
        contentBlobUrl={contentBlobUrl}
        data={data}
        downloadFilename={downloadFilename}
        redirectUrl={redirectUrl}
        requestHeaders={requestHeaders}
        theme={theme}
      />
    );
  };

  return (
    <div className='h-screen bg-white dark:bg-black'>
      <div className={classNames('h-full flex flex-col', [embedded ? 'p-6' : 'px-16px max-w-240 m-auto'])}>
        {!embedded && (
          <div className='h-20 flex items-center gap-12px border-b border-border-secondary'>
            <Button
              aria-label={t('page.preview.buttons.back')}
              icon={<ArrowLeft size={18} />}
              title={t('page.preview.buttons.back')}
              type='text'
              onClick={handleBack}
            />
            <div className='children:h-10'>
              <img
                className='dark:hidden'
                src={logoLight}
              />

              <img
                className='hidden dark:block'
                src={logoDark}
              />
            </div>
          </div>
        )}

        <div
          ref={contentRef}
          className={classNames('flex-1 overflow-hidden', {
            'mt-8': !embedded
          })}
        >
          {renderContent()}
        </div>

        {/* processing panel (W13): lifecycle + chunks for this document —
            hidden in embedded/widget mode, ops actions stay on the ops page */}
        {!embedded && !loading && !error ? (
          <>
            <Button
              type="primary"
              shape="circle"
              icon={<Wrench size={18} />}
              title={t('page.preview.processing.open')}
              className="fixed bottom-24px right-24px z-10"
              onClick={() => setProcessingOpen(true)}
            />
            <Drawer
              title={t('page.preview.processing.title')}
              width={560}
              open={processingOpen}
              onClose={() => setProcessingOpen(false)}
            >
              <ProcessingPanel docId={id} headers={requestHeaders} />
            </Drawer>
          </>
        ) : null}
      </div>
    </div>
  );
}
