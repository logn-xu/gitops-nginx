import React, { useEffect, useState } from "react";
import { Drawer, Button, Badge, Typography, Tag, Space, Descriptions, Alert, Spin,Empty } from "antd";
import { DiffViewer } from "./DiffViewer";

const { Text, Title } = Typography;
const API_BASE = import.meta.env.VITE_API_BASE || "";

interface CommitInfo {
  hash: string;
  message: string;
  author: string;
  timestamp: string;
}

interface GitStatusResponse {
  branch: string;
  sync_mode: string;
  local_commit?: CommitInfo;
  remote_commit?: CommitInfo;
  status: string; // "synced", "ahead", "behind", "diverged", "error"
  diff?: string;
  error?: string;
}

const StatusTag = ({ status }: { status: string }) => {
  switch (status) {
    case "synced":
      return <Tag color="success">已同步</Tag>;
    case "ahead":
      return <Tag color="warning">本地领先 (Ahead)</Tag>;
    case "behind":
      return <Tag color="warning">本地落后 (Behind)</Tag>;
    case "diverged":
      return <Tag color="error">分支冲突 (Diverged)</Tag>;
    case "error":
      return <Tag color="error">错误</Tag>;
    default:
      return <Tag>未知</Tag>;
  }
};

export const GitStatusDrawer: React.FC = () => {
  const [open, setOpen] = useState(false);
  const [data, setData] = useState<GitStatusResponse>();
  const [loading, setLoading] = useState(false);
  const [pollingCount, setPollingCount] = useState(0);

  // Poll status when drawer is open
  useEffect(() => {
    if (!open) return;

    const fetchStatus = () => {
      setLoading(true);
      fetch(`${API_BASE}/api/v1/git/status`)
        .then((res) => res.json())
        .then((data: GitStatusResponse) => {
          setData(data);
        })
        .catch((err) => {
          console.error(err);
        })
        .finally(() => setLoading(false));
    };

    fetchStatus();

    // Poll every 10s if open
    const timer = setInterval(fetchStatus, 10000);
    return () => clearInterval(timer);
  }, [open, pollingCount]);

  const refresh = () => setPollingCount(c => c + 1);

  // Status Indicator logic for the Button
  // We can fetch status periodically even if drawer is closed? 
  // For now, let's keep it simple: Button doesn't auto-update color unless we lift state up or use a context.
  // The requirement says "Add Git Status button ... display dot/color".
  // I will implement a separate small poller for the button or just keep it static until opened.
  // Let's assume user opens it to check.
  
  return (
    <>
      <Button onClick={() => setOpen(true)} style={{ marginRight: 16 }}>
        <Space>
           Git Status
           {data && data.status !== "synced" && (
             <Badge status={data.status === "error" || data.status === "diverged" ? "error" : "warning"} />
           )}
        </Space>
      </Button>
      
      <Drawer
        title="Git 仓库状态"
        width={720}
        onClose={() => setOpen(false)}
        open={open}
        extra={
          <Button onClick={refresh} loading={loading}>
            刷新
          </Button>
        }
      >
        <Spin spinning={loading && !data}>
          {data ? (
            <Space direction="vertical" style={{ width: "100%" }} size="large">
              {data.error && (
                <Alert message="Error" description={data.error} type="error" showIcon />
              )}
              
              <Descriptions bordered column={2}>
                <Descriptions.Item label="分支">{data.branch}</Descriptions.Item>
                <Descriptions.Item label="同步模式">{data.sync_mode}</Descriptions.Item>
                <Descriptions.Item label="状态" span={2}>
                  <StatusTag status={data.status} />
                </Descriptions.Item>
              </Descriptions>

              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
                <div>
                  <Title level={5}>本地 (Local)</Title>
                  {data.local_commit ? (
                    <div style={{ background: "#f5f5f5", padding: 12, borderRadius: 8 }}>
                      <Text strong style={{ display: "block" }}>{data.local_commit.message}</Text>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {data.local_commit.hash.substring(0, 7)} · {data.local_commit.author}
                      </Text>
                      <br/>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {new Date(data.local_commit.timestamp).toLocaleString()}
                      </Text>
                    </div>
                  ) : <Empty description="无信息" />}
                </div>
                <div>
                  <Title level={5}>远程 (Remote)</Title>
                  {data.remote_commit ? (
                    <div style={{ background: "#f5f5f5", padding: 12, borderRadius: 8 }}>
                      <Text strong style={{ display: "block" }}>{data.remote_commit.message}</Text>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {data.remote_commit.hash.substring(0, 7)} · {data.remote_commit.author}
                      </Text>
                      <br/>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {new Date(data.remote_commit.timestamp).toLocaleString()}
                      </Text>
                    </div>
                  ) : <Empty description="无信息" />}
                </div>
              </div>

              {data.diff ? (
                <div>
                  <Title level={5}>差异对比 (Remote vs Local)</Title>
                  <DiffViewer diff={data.diff} maxHeight={500} />
                </div>
              ) : (
                data.status !== "synced" && (
                   <div style={{ textAlign: "center", padding: 20, color: "#999" }}>
                     暂无详细差异信息 (Hash 不同但可能 Diff 计算未涵盖或 Patch 生成失败)
                   </div>
                )
              )}

            </Space>
          ) : (
             !loading && <Empty description="无法获取状态" />
          )}
        </Spin>
      </Drawer>
    </>
  );
};
