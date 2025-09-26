import React from "react";

/**
 * 高亮匹配的文本
 * @param {string} text - 要处理的文本
 * @param {string} query - 搜索查询
 * @param {string} className - 高亮样式类名
 * @returns {React.ReactNode} - 包含高亮的 React 元素
 */
export function highlightText(
  text,
  query,
  className = "highlight-text",
) {
  // 检查输入参数
  if (!text || typeof text !== "string") {
    return text;
  }

  if (!query || typeof query !== "string" || query.trim() === "") {
    return text;
  }

  // 转义特殊正则表达式字符
  const escapedQuery = query.trim().replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

  // 创建不区分大小写的正则表达式
  const regex = new RegExp(`(${escapedQuery})`, "gi");

  // 检查是否有匹配
  if (!regex.test(text)) {
    return text;
  }

  // 重置正则表达式的 lastIndex
  regex.lastIndex = 0;

  // 分割文本并高亮匹配部分
  const parts = text.split(regex);

  return parts.map((part, index) => {
    // 检查这个部分是否匹配查询
    if (part && regex.test(part)) {
      // 重置 lastIndex 以避免状态问题
      regex.lastIndex = 0;
      return (
        <span
          key={index}
          className={className}
          style={{
            backgroundColor: "#fff3cd",
            color: "#856404",
            padding: "2px 6px",
            borderRadius: "4px",
            boxShadow: "0 1px 3px rgba(255, 227, 205, 0.4)",
            display: "inline-block",
            lineHeight: "1.3",
            margin: "0 1px",
            transition: "all 0.2s ease-in-out"
          }}
        >
          {part}
        </span>
      );
    }
    return part;
  });
}

/**
 * 检查文本是否包含查询字符串
 * @param {string} text - 要检查的文本
 * @param {string} query - 搜索查询
 * @returns {boolean} - 是否包含查询
 */
export function containsQuery(text, query) {
  if (
    !text ||
    !query ||
    typeof text !== "string" ||
    typeof query !== "string"
  ) {
    return false;
  }
  return text.toLowerCase().includes(query.toLowerCase());
}
