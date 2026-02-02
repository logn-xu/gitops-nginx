import { Button, Empty, Space, Spin, Typography } from "antd";
import { DiffViewer } from "./DiffViewer";
import type { TripleDiffResponse } from "../types";

const { Title, Text, Paragraph } = Typography;

interface FileContentPanelProps {
  selectedFileKey?: string;
  fileDiff?: TripleDiffResponse;
  fileLoading: boolean;
  isPreviewMode: boolean;
  checkLoading: boolean;
  updateLoading: boolean;
  onCheckConfig: () => void;
  onUpdatePrepare: () => void;
}

export function FileContentPanel({
  selectedFileKey,
  fileDiff,
  fileLoading,
  isPreviewMode,
  checkLoading,
  updateLoading,
  onCheckConfig,
  onUpdatePrepare,
}: FileContentPanelProps) {
  const modeLabel = isPreviewMode ? "预览环境" : "生产环境";

  return (
    <>
      <Space style={{ marginBottom: 12 }}>
        <Button type="primary" loading={checkLoading} onClick={onCheckConfig}>
          执行配置检查
        </Button>
        <Button
          type="primary"
          danger
          loading={updateLoading}
          onClick={onUpdatePrepare}
          disabled={isPreviewMode}
        >
          更新配置
        </Button>
        <Button disabled>预留按钮</Button>
      </Space>
      <Title level={5} style={{ marginTop: 0 }}>
        {selectedFileKey || "选择一个文件查看"}
      </Title>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "1fr 1fr",
          gap: 12,
          minHeight: 360,
        }}
      >
        <div
          style={{
            border: "1px solid #f0f0f0",
            borderRadius: 8,
            padding: 12,
            overflow: "auto",
            gridColumn: "1 / span 2",
          }}
        >
          <Space direction="vertical" style={{ width: "100%" }} size="small">
            <Text strong>远程 Nginx vs {modeLabel} Diff</Text>
            {fileDiff && (
              <Text type="secondary">
                模式: {fileDiff.mode} | 对比: {fileDiff.compare_label}
              </Text>
            )}
            <Spin spinning={fileLoading}>
              {fileDiff ? (
                <DiffViewer diff={fileDiff.diff} emptyText="无差异" />
              ) : (
                <Empty description="未选择文件" />
              )}
            </Spin>
          </Space>
        </div>
        <div
          style={{
            border: "1px solid #f0f0f0",
            borderRadius: 8,
            padding: 12,
            overflow: "auto",
          }}
        >
          <Space direction="vertical" style={{ width: "100%" }} size="small">
            <Text strong>{modeLabel} 文件内容</Text>
            <Spin spinning={fileLoading}>
              {fileDiff ? (
                <Paragraph style={{ whiteSpace: "pre-wrap", fontFamily: "monospace" }}>
                  {fileDiff.compare_content}
                </Paragraph>
              ) : (
                <Empty description="未选择文件" />
              )}
            </Spin>
          </Space>
        </div>
        <div
          style={{
            border: "1px solid #f0f0f0",
            borderRadius: 8,
            padding: 12,
            overflow: "auto",
          }}
        >
          <Space direction="vertical" style={{ width: "100%" }} size="small">
            <Text strong>远程 Nginx 文件内容</Text>
            <Spin spinning={fileLoading}>
              {fileDiff ? (
                <Paragraph style={{ whiteSpace: "pre-wrap", fontFamily: "monospace" }}>
                  {fileDiff.remote_content}
                </Paragraph>
              ) : (
                <Empty description="未选择文件" />
              )}
            </Spin>
          </Space>
        </div>
      </div>
    </>
  );
}
