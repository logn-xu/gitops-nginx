import { Modal, Typography, Tag, Descriptions, Collapse } from "antd";

const { Text, Paragraph } = Typography;

export type CheckResult = {
  ok: boolean;
  mode?: string;
  sync?: {
    total: number;
    skipped: number;
    updated: number;
    deleted: number;
    updated_files?: string[];
    deleted_files?: string[];
  };
  nginx?: {
    command: string;
    ok: boolean;
    output: string;
  };
};

type Props = {
  open: boolean;
  onClose: () => void;
  result?: CheckResult;
};

export function CheckResultModal({ open, onClose, result }: Props) {
  const ok = result?.ok;
  const mode = result?.mode;
  const nginxOk = result?.nginx?.ok;

  return (
    <Modal
      open={open}
      onCancel={onClose}
      onOk={onClose}
      width={860}
      title={
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <span>配置检查结果</span>
          {typeof ok === "boolean" && (
            ok ? <Tag color="green">通过</Tag> : <Tag color="red">失败</Tag>
          )}
          {mode && <Tag>{mode === "preview" ? "预览" : mode}</Tag>}
        </div>
      }
      okText="确认"
      cancelText="关闭"
    >
      {!result ? (
        <Text type="secondary">暂无结果</Text>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <Descriptions size="small" bordered column={1}>
            <Descriptions.Item label="检查状态">
              {ok ? (
                <Text type="success">通过</Text>
              ) : (
                <Text type="danger">失败</Text>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="模式">
              {mode ? (mode === "preview" ? "预览" : mode) : "-"}
            </Descriptions.Item>
            <Descriptions.Item label="同步统计">
              {result.sync ? (
                <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                  <Text>
                    total={result.sync.total} skipped={result.sync.skipped} updated={result.sync.updated} deleted={result.sync.deleted}
                  </Text>
                  {result.sync.updated > 0 && result.sync.updated_files && (
                    <div style={{ fontSize: 12 }}>
                      <Text type="warning">Updated:</Text>
                      <ul style={{ margin: "4px 0", paddingLeft: 20 }}>
                        {result.sync.updated_files.map((f) => (
                          <li key={f}>{f}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                  {result.sync.deleted > 0 && result.sync.deleted_files && (
                    <div style={{ fontSize: 12 }}>
                      <Text type="danger">Deleted:</Text>
                      <ul style={{ margin: "4px 0", paddingLeft: 20 }}>
                        {result.sync.deleted_files.map((f) => (
                          <li key={f}>{f}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              ) : (
                <Text type="secondary">无（生产模式不做同步）</Text>
              )}
            </Descriptions.Item>
          </Descriptions>

          <Collapse
            size="small"
            items={[
              {
                key: "nginx",
                label: (
                  <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                    <span>nginx 检测输出</span>
                    {typeof nginxOk === "boolean" && (
                      nginxOk ? <Tag color="green">OK</Tag> : <Tag color="red">ERROR</Tag>
                    )}
                  </div>
                ),
                children: (
                  <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
                    <div>
                      <Text type="secondary">命令：</Text>
                      <Paragraph style={{ marginBottom: 0, fontFamily: "monospace" }}>
                        {result.nginx?.command || "-"}
                      </Paragraph>
                    </div>
                    <div>
                      <Text type="secondary">输出：</Text>
                      <Paragraph
                        style={{
                          whiteSpace: "pre-wrap",
                          fontFamily: "monospace",
                          marginBottom: 0,
                          maxHeight: 360,
                          overflow: "auto",
                          border: "1px solid #f0f0f0",
                          borderRadius: 6,
                          padding: 8,
                        }}
                      >
                        {result.nginx?.output || ""}
                      </Paragraph>
                    </div>
                  </div>
                ),
              },
            ]}
          />
        </div>
      )}
    </Modal>
  );
}
