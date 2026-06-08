import { Document, Page, pdfjs } from "react-pdf";
import { useState, useEffect, type FC } from "react";
import { Pagination } from "antd";
import { DocDetailProps } from "@/ResultDetail/DocDetail/DocDetail";

pdfjs.GlobalWorkerOptions.workerSrc = `https://unpkg.com/pdfjs-dist@${pdfjs.version}/build/pdf.worker.min.mjs`;

interface PdfProps extends DocDetailProps {
  url: string;
}

const Pdf: FC<PdfProps> = (props) => {
  const { url, requestHeaders } = props;

  const [numPages, setNumPages] = useState(0);
  const [pageNumber, setPageNumber] = useState(1);
  const [fileData, setFileData] = useState<{ data: ArrayBuffer } | string>(url);

  useEffect(() => {
    if (requestHeaders) {
      fetch(url, { headers: requestHeaders })
        .then((res) => res.arrayBuffer())
        .then((data) => setFileData({ data }));
    } else {
      setFileData(url);
    }
  }, [url, requestHeaders]);

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

      <div className="border border-solid border-border rounded-lg overflow-hidden">
        <Document
          file={fileData}
          onLoadSuccess={(pdf) => {
            setNumPages(pdf.numPages);
          }}
        >
          <Page
            className="children:(w-full! h-unset!)"
            pageNumber={pageNumber}
            renderTextLayer={false}
            renderAnnotationLayer={false}
          />
        </Document>
      </div>
    </div>
  );
};

export default Pdf;
