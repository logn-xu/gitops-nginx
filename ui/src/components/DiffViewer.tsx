import React from "react";

export type DiffViewerProps = {
  diff?: string;
  emptyText?: string;
  maxHeight?: number;
};

function getLineStyle(line: string): React.CSSProperties {
  if (line.startsWith("+++ ") || line.startsWith("--- ")) {
    return { color: "#8c8c8c" };
  }
  if (line.startsWith("@@")) {
    return { background: "#fff7e6", color: "#ad6800" };
  }
  if (line.startsWith("+")) {
    return { background: "#f6ffed", color: "#389e0d" };
  }
  if (line.startsWith("-")) {
    return { background: "#fff1f0", color: "#cf1322" };
  }
  return {};
}

export function DiffViewer({ diff, emptyText = "无差异", maxHeight = 420 }: DiffViewerProps) {
  if (!diff || diff.trim() === "") {
    return (
      <pre
        style={{
          whiteSpace: "pre-wrap",
          fontFamily: "monospace",
          margin: 0,
          padding: 8,
          border: "1px solid #f0f0f0",
          borderRadius: 6,
          background: "#fafafa",
        }}
      >
        {emptyText}
      </pre>
    );
  }

  const lines = diff.replace(/\r\n/g, "\n").split("\n");

  return (
    <div
      style={{
        border: "1px solid #f0f0f0",
        borderRadius: 6,
        background: "#fafafa",
        maxHeight,
        overflow: "auto",
      }}
    >
      <pre
        style={{
          whiteSpace: "pre",
          fontFamily: "monospace",
          margin: 0,
          padding: 8,
        }}
      >
        {lines.map((line, idx) => (
          <div key={idx} style={{ ...getLineStyle(line), padding: "0 4px" }}>
            {line === "" ? " " : line}
          </div>
        ))}
      </pre>
    </div>
  );
}
