import { Document, Page, pdfjs } from "react-pdf";
import { useState, useEffect, type FC } from "react";
import { Pagination } from "antd";
import { DocDetailProps } from "@/ResultDetail/DocDetail/DocDetail";

pdfjs.GlobalWorkerOptions.workerSrc = `https://unpkg.com/pdfjs-dist@${pdfjs.version}/build/pdf.worker.min.mjs`;

interface PdfProps extends DocDetailProps {
  url: string;
  onLoadingChange?: (loading: boolean) => void;
}

const Pdf: FC<PdfProps> = (props) => {
  const { url, requestHeaders, onLoadingChange } = props;

  const [numPages, setNumPages] = useState(0);
  const [pageNumber, setPageNumber] = useState(1);
  
  const [pdfUrl, setPdfUrl] = useState<string>("");

  useEffect(() => {
    let isCurrent = true;
    let generatedBlobUrl = "";

    if (requestHeaders && url) {
      onLoadingChange?.(true);
      
      fetch(url, { headers: requestHeaders })
        .then((res) => res.blob()) 
        .then((blob) => {
          if (!isCurrent) return;
          
          generatedBlobUrl = URL.createObjectURL(blob);
          setPdfUrl(generatedBlobUrl);
        })
        .catch((err) => {
        })
        .finally(() => {
          if (isCurrent) onLoadingChange?.(false);
        });
    } else {
      setPdfUrl(url);
    }

    return () => {
      isCurrent = false;
      if (generatedBlobUrl) {
        URL.revokeObjectURL(generatedBlobUrl); 
      }
    };
  }, [url, requestHeaders]);

  useEffect(() => {
    setPageNumber(1);
  }, [pdfUrl]);

  return (
    <div className="flex flex-col gap-2">
      <div className="flex justify-end">
        <Pagination
          size="small"
          pageSize={1}
          total={numPages}
          current={pageNumber}
          showSizeChanger={false}
          onChange={(page) => setPageNumber(page)}
        />
      </div>

      <div className="border border-[#F0F0F0] dark:border-[#303030] rounded-lg overflow-hidden">
        {pdfUrl && (
          <Document
            file={pdfUrl}
            onLoadSuccess={(pdf) => {
              setNumPages(pdf.numPages);
            }}
            onLoadError={(err) => console.error("PDF load error:", err)}
          >
            <Page
              className="children:(w-full! h-unset!)"
              pageNumber={pageNumber}
              renderTextLayer={false}
              renderAnnotationLayer={false}
            />
          </Document>
        )}
      </div>
    </div>
  );
};

export default Pdf;