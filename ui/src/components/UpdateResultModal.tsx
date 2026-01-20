import { Modal, Typography, Tag, Descriptions, Collapse } from "antd";

const { Text, Paragraph } = Typography;

type CheckResult = {
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

export type UpdatePrepareResponse = {
  success: boolean;
  nginx?: {
    command: string;
    ok: boolean;
    output: string;
  };
  sync?: {
    total: number;
    skipped: number;
    updated: number;
    added: number;
    deleted: number;
    updated_files?: string[];
    added_files?: string[];
    deleted_files?: string[];
  };
};

export type UpdateApplyResponse = {
  success: boolean;
  message: string;
  nginx?: {
    command: string;
    ok: boolean;
    output: string;
  };
};

type Props = {
  open: boolean;
  mode: "prepare" | "apply";
  loading?: boolean;
  onCancel: () => void;
  onConfirmApply: () => void;
  data?: UpdatePrepareResponse | UpdateApplyResponse;
};

export function UpdateResultModal({
  open,
  mode,
  loading,
  onCancel,
  onConfirmApply,
  data,
}: Props) {
  const isPrepare = mode === "prepare";

  // Handle both prepare and apply responses
  const prepareData = data as UpdatePrepareResponse | undefined;
  const applyData = data as UpdateApplyResponse | undefined;

  // For prepare: success comes from nginx.ok
  // For apply: success comes from success field
  const ok = isPrepare ? prepareData?.nginx?.ok : applyData?.success;
  const sync = prepareData?.sync;
  const nginx = isPrepare ? prepareData?.nginx : applyData?.nginx;
  const nginxOk = nginx?.ok;
  const message = applyData?.message;

  const confirmDisabled = !ok || loading;

  return (
    <Modal
      open={open}
      onCancel={onCancel}
      onOk={isPrepare ? onConfirmApply : onCancel}
      confirmLoading={loading}
      okText={isPrepare ? "确认并更新(执行 reload)" : "关闭"}
      cancelText={isPrepare ? "取消" : "关闭"}
      width={920}
      okButtonProps={{ disabled: confirmDisabled }}
      title={
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <span>更新配置结果</span>
          {typeof ok === "boolean" && (
            ok ? <Tag color="green">检查通过</Tag> : <Tag color="red">检查失败</Tag>
          )}
        </div>
      }
    >
      {!data ? (
        <Text type="secondary">暂无结果</Text>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <Descriptions size="small" bordered column={1}>
            <Descriptions.Item label="阶段">
              {isPrepare ? "准备阶段(同步+检查)" : "应用阶段(同步+reload)"}
            </Descriptions.Item>
            <Descriptions.Item label="检查状态">
              {ok ? <Text type="success">通过</Text> : <Text type="danger">失败</Text>}
            </Descriptions.Item>
            {sync && (
              <Descriptions.Item label="同步统计">
                <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                  <Text>
                    total={sync.total} skipped={sync.skipped} added={sync.added} updated={sync.updated} deleted={sync.deleted}
                  </Text>
                  {sync.added > 0 && sync.added_files && (
                    <div style={{ fontSize: 12 }}>
                      <Text type="success">Added:</Text>
                      <ul style={{ margin: "4px 0", paddingLeft: 20 }}>
                        {sync.added_files.map((f) => (
                          <li key={f}>{f}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                  {sync.updated > 0 && sync.updated_files && (
                    <div style={{ fontSize: 12 }}>
                      <Text type="warning">Updated:</Text>
                      <ul style={{ margin: "4px 0", paddingLeft: 20 }}>
                        {sync.updated_files.map((f) => (
                          <li key={f}>{f}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                  {sync.deleted > 0 && sync.deleted_files && (
                    <div style={{ fontSize: 12 }}>
                      <Text type="danger">Deleted:</Text>
                      <ul style={{ margin: "4px 0", paddingLeft: 20 }}>
                        {sync.deleted_files.map((f) => (
                          <li key={f}>{f}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              </Descriptions.Item>
            )}
            {!isPrepare && message && (
              <Descriptions.Item label="消息">
                <Text>{message}</Text>
              </Descriptions.Item>
            )}
          </Descriptions>

          <Collapse
            size="small"
            items={[
              {
                key: "nginx",
                label: (
                  <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                    <span>{isPrepare ? "nginx -t 检测输出" : "nginx reload 输出"}</span>
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
                        {nginx?.command || "-"}
                      </Paragraph>
                    </div>
                    <div>
                      <Text type="secondary">输出：</Text>
                      <Paragraph
                        style={{
                          whiteSpace: "pre-wrap",
                          fontFamily: "monospace",
                          marginBottom: 0,
                          maxHeight: 320,
                          overflow: "auto",
                          border: "1px solid #f0f0f0",
                          borderRadius: 6,
                          padding: 8,
                        }}
                      >
                        {nginx?.output || ""}
                      </Paragraph>
                    </div>
                  </div>
                ),
              },
            ]}
          />

          {isPrepare && !ok && (
            <Text type="danger">检查未通过，无法执行更新(reload)。</Text>
          )}
        </div>
      )}
    </Modal>
  );
}
