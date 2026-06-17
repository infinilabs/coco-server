import { Document, Page, pdfjs } from "react-pdf";
import { useState, useEffect, useRef, type FC } from "react";
import { Pagination } from "antd";
import { DocDetailProps } from "@/ResultDetail/DocDetail/DocDetail";

pdfjs.GlobalWorkerOptions.workerSrc = `https://unpkg.com/pdfjs-dist@${pdfjs.version}/build/pdf.worker.min.mjs`;

interface PdfProps extends DocDetailProps {
  url: string;
  onLoadingChange?: (loading: boolean) => void;
  onLoadError?: (error: any) => void;
  className?: string;
}

const Pdf: FC<PdfProps> = (props) => {
  const { url, requestHeaders, onLoadingChange, onLoadError, className = '' } = props;

  const [numPages, setNumPages] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [pdfUrl, setPdfUrl] = useState<string>("");

  const containerRef = useRef<HTMLDivElement>(null);
  const pageRefs = useRef<Map<number, HTMLDivElement>>(new Map());

  useEffect(() => {
    let isCurrent = true;
    let generatedBlobUrl = "";

    if (url) {
      onLoadingChange?.(true);

      fetch(url, { headers: requestHeaders })
        .then((res) => {
          if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
          return res.blob();
        })
        .then((blob) => {
          if (!isCurrent) return;
          generatedBlobUrl = URL.createObjectURL(blob);
          setPdfUrl(generatedBlobUrl);
        })
        .catch((err) => {
          onLoadingChange?.(false);
          onLoadError?.(err);
          setPdfUrl("");
        })
    }

    return () => {
      isCurrent = false;
      if (generatedBlobUrl) {
        URL.revokeObjectURL(generatedBlobUrl);
      }
    };
  }, [url, JSON.stringify(requestHeaders)]);

  useEffect(() => {
    setCurrentPage(1);
    pageRefs.current.clear();
  }, [pdfUrl]);

  useEffect(() => {
    if (numPages === 0) return;

    const visiblePages = new Map<number, number>();

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          const pageNum = Number(entry.target.getAttribute('data-page-num'));
          if (!pageNum) return;

          if (entry.isIntersecting) {
            visiblePages.set(pageNum, entry.intersectionRatio);
          } else {
            visiblePages.delete(pageNum);
          }
        });

        if (visiblePages.size > 0) {
          let maxRatio = -1;
          let topPage = 1;
          visiblePages.forEach((ratio, pageNum) => {
            if (ratio > maxRatio) {
              maxRatio = ratio;
              topPage = pageNum;
            }
          });
          setCurrentPage(topPage);
        }
      },
      {
        root: containerRef.current,
        threshold: [0, 0.25, 0.5, 0.75, 1],
      }
    );

    pageRefs.current.forEach((el) => {
      observer.observe(el);
    });

    return () => {
      observer.disconnect();
    };
  }, [numPages]);

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    const pageElement = pageRefs.current.get(page);
    if (pageElement && containerRef.current) {
      const containerTop = containerRef.current.getBoundingClientRect().top;
      const elementTop = pageElement.getBoundingClientRect().top;
      containerRef.current.scrollTop += elementTop - containerTop - 8;
    }
  };

  return (
    <div className={`flex flex-col gap-2 h-full ${className}`}>
      <div className="flex items-center justify-end px-24px">
        <Pagination
          simple
          pageSize={1}
          total={numPages}
          current={currentPage}
          showSizeChanger={false}
          onChange={handlePageChange}
          classNames={{
            root: "[&_.ant-pagination-prev]:!hidden [&_.ant-pagination-next]:!hidden",
          }}
        />
      </div>

      <div
        ref={containerRef}
        className="pt-8px pb-24px px-24px overflow-y-auto w-full h-full flex flex-col gap-4 items-center rounded-lg"
      >
        {pdfUrl && (
          <Document
            file={pdfUrl}
            onLoadSuccess={(pdf) => {
              setNumPages(pdf.numPages);
              onLoadingChange?.(false);
            }}
            onLoadError={(err) => {
              onLoadingChange?.(false);
              onLoadError?.(err);
              setPdfUrl("");
            }}
            className="flex flex-col gap-4 w-full items-center"
          >
            {Array.from(new Array(numPages), (_, index) => {
              const pageNum = index + 1;
              return (
                <div
                  key={pageNum}
                  data-page-num={pageNum}
                  ref={(el) => {
                    if (el) {
                      pageRefs.current.set(pageNum, el);
                    } else {
                      pageRefs.current.delete(pageNum);
                    }
                  }}
                  className="shadow-[0_0_12px_rgba(0,0,0,0.1)] rounded w-full max-w-full"
                >
                  <Page
                    className="children:(w-full! h-unset!)"
                    pageNumber={pageNum}
                    renderTextLayer={false}
                    renderAnnotationLayer={false}
                  />
                </div>
              );
            })}
          </Document>
        )}
      </div>
    </div>
  );
};

export default Pdf;
