import { Button, Result, Spin } from 'antd';
import { useParams } from 'react-router-dom';

import { request } from '@/service/request';
import { fetchEntityUser } from '@/service/api/entity';
import logoLight from '@/assets/imgs/coco-logo-text-light.svg';
import logoDark from '@/assets/imgs/coco-logo-text-dark.svg';
import { getDarkMode } from '@/store/slice/theme';
import { useAppSelector } from '@/hooks/business/useStore';
import classNames from 'classnames';

import PreviewContent from './components/PreviewContent';

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
  const { t } = useTranslation();

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
      const blobUrl = URL.createObjectURL(blob);
      const filename = parseFilenameFromContentDisposition(
        res.headers.get('Content-Disposition'),
        docData.title || 'download'
      );

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

  const renderContent = () => {
    if (loading) {
      return (
        <Spin
          fullscreen
          percent='auto'
          spinning={loading}
        />
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
          <div className='h-20 flex items-center border-b border-border-secondary'>
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
          className={classNames('flex-1 overflow-hidden', {
            'mt-8': !embedded
          })}
        >
          {renderContent()}
        </div>
      </div>
    </div>
  );
}
