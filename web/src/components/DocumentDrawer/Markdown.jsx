import React, { useRef, useState, useEffect, useMemo, forwardRef } from "react";
import clsx from "clsx";
import { Segmented } from "antd";
import ReactMarkdown from "react-markdown";
import RemarkMath from "remark-math";
import RemarkBreaks from "remark-breaks";
import RehypeKatex from "rehype-katex";
import RemarkGfm from "remark-gfm";
import RehypeHighlight from "rehype-highlight";
import RehypeRaw from "rehype-raw";
import mermaid from "mermaid";
import { useDebouncedCallback } from "use-debounce";
import { useTranslation } from "react-i18next";

import ProgressiveImage from "../common/ProgressiveImage";

import "./markdown.scss";
import "./highlight.css";

// 8
// NOTE: 本文件是 .jsx（非 TSX），不能用 useRef<HTMLDivElement>(null) 泛型写法 ——
// esbuild 会把泛型解析成比较表达式 `useRef < HTMLDivElement > null`（= false），
// 导致 ref 永远无效。此处沿用上游 NextChat 的 TSX 代码移植时遗留的坑。
function Mermaid(props) {
  const ref = useRef(null);
  const [hasError, setHasError] = useState(false);
  const [view, setView] = useState("diagram"); // diagram | code，默认图示
  const { t } = useTranslation();

  useEffect(() => {
    if (props.code && ref.current) {
      mermaid
        .run({
          nodes: [ref.current],
          suppressErrors: true,
        })
        .then(() => {
          // suppressErrors 时语法错误不 reject，而是插入 aria-roledescription="error" 的错误图
          const svg = ref.current?.querySelector("svg");
          const isErr = !svg || svg.getAttribute("aria-roledescription") === "error";
          setHasError(isErr);
          if (isErr) console.error("[Mermaid] diagram syntax error, kept as source");
        })
        .catch((e) => {
          setHasError(true);
          console.error("[Mermaid] ", e.message);
        });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.code]);

  // 默认展示渲染后的图，Segmented 可在 图示/源码 间切换（与全局 Segmented 风格一致）
  // 标题取自图表类型关键字（代码首词），如 mindmap → Mindmap
  // NOTE: 必须保持在下方 hasError early-return 之前 —— hooks 数量在两次渲染
  // 间必须一致，否则 React 报 "Rendered fewer hooks than expected"
  const diagramTitle = useMemo(() => {
    const m = (props.code || "").trimStart().match(/^[a-zA-Z]+/);
    if (!m) return "Diagram";
    const w = m[0];
    return w.charAt(0).toUpperCase() + w.slice(1);
  }, [props.code]);

  if (hasError) {
    // mindmap 等图表语法对缩进敏感，LLM 输出解析失败时保留原始代码并给出可见提示，
    // 不再静默隐藏（此前 hasError 直接 return null，用户只会看到图凭空消失）
    return (
      <div className={clsx("no-dark", "mermaid", "mermaid-error")}>
        <div className="mermaid-error-banner">
          {t("common.mermaidRenderError")}
        </div>
        <pre className="mermaid-error-code">{props.code}</pre>
      </div>
    );
  }

  return (
    <div className={clsx("no-dark", "mermaid", "mermaid-block")}>
      <div className="mermaid-toolbar">
        <span className="mermaid-title">{diagramTitle}</span>
        <Segmented
          size="small"
          value={view}
          onChange={(v) => setView(v)}
          options={[
            { label: t("common.diagram"), value: "diagram" },
            { label: t("common.sourceCode"), value: "code" },
          ]}
        />
      </div>
      <div
        className="mermaid-view-diagram"
        style={{ display: view === "diagram" ? "block" : "none" }}
        ref={ref}
      >
        {props.code}
      </div>
      <pre
        className="mermaid-view-code"
        style={{ display: view === "code" ? "block" : "none" }}
      >
        {props.code}
      </pre>
    </div>
  );
}

// 7
// NOTE: react-markdown v10 (React 18) 不会向自定义组件转发 ref —— 此前的
// forwardRef ref 恒为 null，mermaid 提取从未执行（图表永远停留在代码块）。
// 改用组件内部 ref 挂真实 <pre> 节点；effect 依赖 props.children，
// 内容变化（含流式输出）时也会重新扫描。
const PreCode = forwardRef((props, _ref) => {

  const preRef = useRef(null);
  const [mermaidCode, setMermaidCode] = useState("");
  // const [htmlCode, setHtmlCode] = useState("");
  // const { height } = useWindowSize();
  // console.log(htmlCode, height);

  const renderArtifacts = useDebouncedCallback(() => {
    if (!preRef?.current) return;
    const mermaidDom = preRef.current.querySelector("code.language-mermaid");
    if (mermaidDom) {
      setMermaidCode(mermaidDom.innerText);
    }
    // const htmlDom = ref.current.querySelector("code.language-html");
    // const refText = ref.current.querySelector("code")?.innerText;
    // if (htmlDom) {
    //   setHtmlCode((htmlDom as HTMLElement).innerText);
    // } else if (refText?.startsWith("<!DOCTYPE")) {
    //   setHtmlCode(refText);
    // }
  }, 600);

  // const enableArtifacts = true;
  // console.log(enableArtifacts);

  //Wrap the paragraph for plain-text
  useEffect(() => {
    if (preRef?.current) {
      const codeElements = preRef.current.querySelectorAll("code");
      const wrapLanguages = [
        "",
        "md",
        "markdown",
        "text",
        "txt",
        "plaintext",
        "tex",
        "latex",
      ];
      codeElements.forEach((codeElement) => {
        let languageClass = codeElement.className.match(/language-(\w+)/);
        let name = languageClass ? languageClass[1] : "";
        if (wrapLanguages.includes(name)) {
          codeElement.style.whiteSpace = "pre-wrap";
        }
      });
      setTimeout(renderArtifacts, 1);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.children]);

  return (
    <>
      <pre ref={preRef} style={mermaidCode ? { display: "none" } : undefined}>
        <span
          className="copy-code-button"
          onClick={() => {
            if (preRef?.current) {
              //todo copyToClipboard
            }
          }}
        ></span>
        {props.children}
      </pre>
      {mermaidCode.length > 0 && (
        <Mermaid code={mermaidCode} key={mermaidCode} />
      )}
    </>
  );
})

// 6
const CustomCode = forwardRef((props, ref) => {
  const enableCodeFold = false;

  const [collapsed, setCollapsed] = useState(true);
  const [showToggle, setShowToggle] = useState(false);

  useEffect(() => {
    if (ref?.current) {
      const codeHeight = ref.current.scrollHeight;
      setShowToggle(codeHeight > 400);
      ref.current.scrollTop = ref.current.scrollHeight;
    }
  }, [ref?.current]);

  const toggleCollapsed = () => {
    setCollapsed((collapsed) => !collapsed);
  };
  const renderShowMoreButton = () => {
    if (showToggle && enableCodeFold && collapsed) {
      return (
        <div
          className={clsx("show-hide-button", {
            collapsed,
            expanded: !collapsed,
          })}
        >
          <button onClick={toggleCollapsed}>{"NewChat More"}</button>
        </div>
      );
    }
    return null;
  };
  return (
    <>
      <code
        className={clsx(props?.className)}
        ref={ref}
        style={{
          maxHeight: enableCodeFold && collapsed ? "400px" : "none",
          overflowY: "hidden",
        }}
      >
        {props.children}
      </code>

      {renderShowMoreButton()}
    </>
  );
})

// 5
function escapeBrackets(text) {
  if (!text) return text;
  const pattern =
    /(```[\s\S]*?```|`.*?`)|\\\[([\s\S]*?[^\\])\\\]|\\\((.*?)\\\)/g;
  return text?.replace(
    pattern,
    (match, codeBlock, squareBracket, roundBracket) => {
      if (codeBlock) {
        return codeBlock;
      } else if (squareBracket) {
        return `$$${squareBracket}$$`;
      } else if (roundBracket) {
        return `$${roundBracket}$`;
      }
      return match;
    }
  );
}

// 4
function tryWrapHtmlCode(text) {
  // try add wrap html code (fixed: html codeblock include 2 newline)
  if (!text) return text;
  return text
    ?.replace(
      /([`]*?)(\w*?)([\n\r]*?)(<!DOCTYPE html>)/g,
      (match, quoteStart, doctype) => {
        return !quoteStart ? "\n```html\n" + doctype : match;
      }
    )
    ?.replace(
      /(<\/body>)([\r\n\s]*?)(<\/html>)([\n\r]*)([`]*)([\n\r]*?)/g,
      (match, bodyEnd, space, htmlEnd, quoteEnd) => {
        return !quoteEnd ? bodyEnd + space + htmlEnd + "\n```\n" : match;
      }
    );
}

// 3
function _MarkDownContent(props) {
  const escapedContent = useMemo(() => {
    return tryWrapHtmlCode(escapeBrackets(props.content));
  }, [props.content]);

  // renderLink, when provided, gets first refusal on every link — returning
  // anything non-undefined replaces the anchor entirely (wiki capsules).
  const renderLink = props.renderLink;

  return (
    <ReactMarkdown
      remarkPlugins={[RemarkMath, RemarkGfm, RemarkBreaks]}
      rehypePlugins={[
        RehypeRaw,
        RehypeKatex,
        [
          RehypeHighlight,
          {
            detect: false,
            ignoreMissing: true,
          },
        ],
      ]}
      components={{
        pre: PreCode,
        code: CustomCode,
        p: (pProps) => <p {...pProps} dir="auto" />,
        // markdown images arrive without dimensions and without error
        // handling — route them through the progressive tile so loading and
        // broken sources both look designed instead of browser-default
        img: (imgProps) => {
          // wiki 约定:alt 尾部 "|像素" 指定显示宽度(编辑器尺寸工具条写入)
          const widthFromAlt = (imgProps.alt || '').match(/\|(\d{2,4})$/);
          const style = widthFromAlt ? { width: `${widthFromAlt[1]}px`, maxWidth: '100%', height: 'auto' } : undefined;
          return <ProgressiveImage {...imgProps} className={imgProps.className} style={style} />;
        },
        a: (aProps) => {
          if (renderLink) {
            const replaced = renderLink(aProps.href || "", aProps.children);
            if (replaced !== undefined) return replaced;
          }
          const href = aProps.href || "";
          if (/\.(aac|mp3|opus|wav)$/.test(href)) {
            return (
              <figure>
                <audio controls src={href}></audio>
              </figure>
            );
          }
          if (/\.(3gp|3g2|webm|ogv|mpeg|mp4|avi)$/.test(href)) {
            return (
              <video controls width="99.9%">
                <source src={href} />
              </video>
            );
          }
          const isInternal = /^\/#/i.test(href);
          const target = isInternal ? "_self" : aProps.target ?? "_blank";
          return <a {...aProps} target={target} />;
        },
      }}
    >
      {escapedContent}
    </ReactMarkdown>
  );
}

// 2
const MarkdownContent = React.memo(_MarkDownContent);

// 1
export default function Markdown(props) {
  const mdRef = useRef(null);

  return (
    <div className="coco-chat">
      <div
        className="markdown-body"
        style={{
          fontSize: `${props.fontSize ?? 14}px`,
          fontFamily: props.fontFamily || "inherit",
        }}
        ref={mdRef}
        onContextMenu={props.onContextMenu}
        onDoubleClickCapture={props.onDoubleClickCapture}
        dir="auto"
      >
        { props.content ? <MarkdownContent content={props.content} renderLink={props.renderLink} /> : null}
      </div>
    </div>
  );
}
