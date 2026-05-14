import React from "react";

export function highlightText(
  text,
  query,
  className = "highlight-text",
) {
  if (!text || typeof text !== "string") {
    return text;
  }

  if (!query || typeof query !== "string" || query.trim() === "") {
    return text;
  }

  const escapedQuery = query.trim().replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

  const regex = new RegExp(`(${escapedQuery})`, "gi");

  if (!regex.test(text)) {
    return text;
  }

  regex.lastIndex = 0;

  const parts = text.split(regex);

  return parts.map((part, index) => {
    if (part && regex.test(part)) {
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
