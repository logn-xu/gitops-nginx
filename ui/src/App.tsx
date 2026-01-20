import React, { Key, useEffect, useMemo, useState } from "react";
import {
  App as AntdApp,
  Button,
  Empty,
  Layout,
  Menu,
  Space,
  Spin,
  Switch,
  Typography,
} from "antd";
import type { MenuProps, TreeDataNode } from "antd";
import { Tree } from "antd";
import { AutoRefresh } from "./components/AutoRefresh";
import { GitStatusDrawer } from "./components/GitStatusDrawer";
import { CheckResultModal, type CheckResult } from "./components/CheckResultModal";
import { DiffViewer } from "./components/DiffViewer";
import {
  UpdateResultModal,
  type UpdateApplyResponse,
  type UpdatePrepareResponse,
} from "./components/UpdateResultModal";

type GroupSummary = {
  name: string;
  hosts: { name: string; host: string; config_dir_suffix: string }[];
};

type GroupsResponse = {
  groups: GroupSummary[];
};

type TreeResponse = {
  prefix: string;
  paths: string[];
  diff_paths?: string[];
  file_statuses?: Record<string, string>;
};

type TripleDiffResponse = {
  path: string;
  remote_content: string;
  compare_content: string;
  diff: string;
  mode: string;
  compare_label: string;
  file_status?: string;
};

const { Header, Sider, Content } = Layout;
const { Title, Text, Paragraph } = Typography;
const API_BASE = import.meta.env.VITE_API_BASE || "";

const STATUS_MARKERS: Record<string, { icon: string; color: string; label: string }> = {
  modified: { icon: "★", color: "#faad14", label: "修改" },
  added: { icon: "+", color: "#52c41a", label: "新增" },
  deleted: { icon: "-", color: "#ff4d4f", label: "删除" },
};

function buildTree(
  prefix: string,
  paths: string[],
  fileStatuses: Record<string, string>,
  showAll: boolean
): TreeDataNode[] {
  const root: Record<string, any> = {};

  // 1. Construct raw hierarchy
  paths.forEach((p) => {
    const segments = p.split("/").filter(Boolean);
    let current = root;
    segments.forEach((seg, idx) => {
      const isLeaf = idx === segments.length - 1;
      const relPath = segments.slice(0, idx + 1).join("/");
      const status = fileStatuses[relPath];

      if (!current[seg]) {
        current[seg] = {
          rawTitle: seg,
          key: prefix ? `${prefix}/${relPath}` : relPath,
          isLeaf,
          children: isLeaf ? undefined : {},
          status,
        };
      }
      current = (current[seg] as any).children ?? {};
    });
  });

  // 2. Recursive process: Filter & Sort
  const process = (
    nodesMap: Record<string, any>,
  ): { nodes: TreeDataNode[]; hasChange: boolean } => {
    const resultNodes: any[] = [];
    let groupHasChange = false;

    Object.values(nodesMap).forEach((node) => {
      let childHasChange = false;
      let children: TreeDataNode[] | undefined = undefined;

      if (!node.isLeaf && node.children) {
        const childResult = process(node.children);
        children = childResult.nodes;
        childHasChange = childResult.hasChange;
      }

      // A node "has change" if it's a modified file OR contains modified files
      const nodeHasChange = !!node.status || childHasChange;

      if (nodeHasChange) {
        groupHasChange = true;
      }

      // Filter logic:
      // If we only show modified (showAll = false), drop nodes that have no change.
      if (!showAll && !nodeHasChange) {
        return;
      }

      // Construct Title with Icon if needed
      const marker = node.status ? STATUS_MARKERS[node.status] : null;
      const title =
        node.isLeaf && marker ? (
          <span>
            <span style={{ color: marker.color, marginRight: 4 }}>
              {marker.icon}
            </span>
            {node.rawTitle}
          </span>
        ) : (
          node.rawTitle
        );

      resultNodes.push({
        title,
        key: node.key,
        isLeaf: node.isLeaf,
        children,
        _isModified: nodeHasChange, // internal flag for sorting
        _rawTitle: node.rawTitle,
      });
    });

    // Sort: Modified first, then Alphabetical
    resultNodes.sort((a, b) => {
      if (a._isModified !== b._isModified) {
        return a._isModified ? -1 : 1;
      }
      return a._rawTitle.localeCompare(b._rawTitle);
    });

    return { nodes: resultNodes, hasChange: groupHasChange };
  };

  return process(root).nodes;
}

function App() {
  const { message: antMessage } = AntdApp.useApp();
  const [collapsedNav, setCollapsedNav] = useState(false);
  const [groups, setGroups] = useState<GroupSummary[]>([]);
  const [selectedGroup, setSelectedGroup] = useState<string>();
  const [selectedHost, setSelectedHost] = useState<string>();
  const [selectedConfigSuffix, setSelectedConfigSuffix] = useState<string>();
  const [isPreviewMode, setIsPreviewMode] = useState(true);
  const [showAllFiles, setShowAllFiles] = useState(true);

  // const [treePrefix, setTreePrefix] = useState<string>("");
  const [treeData, setTreeData] = useState<TreeDataNode[]>([]);
  const [treeKey, setTreeKey] = useState(0);
  const [treeLoading, setTreeLoading] = useState(false);
  // const [fileStatuses, setFileStatuses] = useState<Record<string, string>>({});

  const [selectedFileKey, setSelectedFileKey] = useState<string>();
  const [fileLoading, setFileLoading] = useState(false);
  const [fileDiff, setFileDiff] = useState<TripleDiffResponse>();
  const [checkLoading, setCheckLoading] = useState(false);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const handleRefresh = () => {
    setRefreshTrigger((prev) => prev + 1);
  };

  const [checkModalOpen, setCheckModalOpen] = useState(false);
  const [checkResult, setCheckResult] = useState<CheckResult>();

  const [updateLoading, setUpdateLoading] = useState(false);
  const [updateModalOpen, setUpdateModalOpen] = useState(false);
  const [updateStage, setUpdateStage] = useState<"prepare" | "apply">("prepare");
  const [updateResult, setUpdateResult] = useState<
    UpdatePrepareResponse | UpdateApplyResponse
  >();

  useEffect(() => {
    fetch(`${API_BASE}/api/v1/groups`)
      .then((res) => res.json())
      .then((data: GroupsResponse) => {
        setGroups(data.groups || []);
        if (data.groups?.[0]) {
          const firstGroup = data.groups[0];
          const firstHost = firstGroup.hosts?.[0];
          if (firstGroup && firstHost) {
            setSelectedGroup(firstGroup.name);
            setSelectedHost(firstHost.host);
            setSelectedConfigSuffix(firstHost.config_dir_suffix);
          }
        }
      })
      .catch((err) => {
        console.error(err);
        antMessage.error("获取分组列表失败");
      });
  }, [antMessage]);

  useEffect(() => {
    setSelectedFileKey(undefined);
    setFileDiff(undefined);
  }, [selectedGroup, selectedHost, isPreviewMode]);

  useEffect(() => {
    if (!selectedGroup || !selectedHost) return;
    setTreeLoading(true);
    fetch(
      `${API_BASE}/api/v1/tree?group=${encodeURIComponent(
        selectedGroup,
      )}&host=${encodeURIComponent(selectedHost)}&mode=${isPreviewMode ? "preview" : "prod"}`,
    )
      .then((res) => res.json())
      .then((data: TreeResponse) => {
        const statuses = data.file_statuses || {};
        setTreeData(buildTree(data.prefix, data.paths || [], statuses, showAllFiles));
        setTreeKey((k) => k + 1);
      })
      .catch((err) => {
        console.error(err);
        antMessage.error("获取文件树失败");
      })
      .finally(() => setTreeLoading(false));
  }, [selectedGroup, selectedHost, isPreviewMode, antMessage, refreshTrigger, showAllFiles]);

  const handleSelectFile = (keys: Key[]) => {
    const key = keys[0] as string | undefined;
    if (!key || !selectedGroup || !selectedHost) return;
    setSelectedFileKey(key);
    setFileLoading(true);
    fetch(
      `${API_BASE}/api/v1/triple-diff?group=${encodeURIComponent(
        selectedGroup,
      )}&host=${encodeURIComponent(selectedHost)}&path=${encodeURIComponent(
        key,
      )}&mode=${isPreviewMode ? "preview" : "prod"}`,
    )
      .then((res) => {
        if (!res.ok) throw new Error("failed to fetch file diff");
        return res.json();
      })
      .then((data: TripleDiffResponse) => {
        setFileDiff(data);
      })
      .catch((err) => {
        console.error(err);
        antMessage.error("获取文件内容/差异失败");
      })
      .finally(() => setFileLoading(false));
  };

  useEffect(() => {
    if (!selectedFileKey || !selectedGroup || !selectedHost) return;
    
    // Silent refresh (or with loading if desired, but silent is better for auto-refresh usually? 
    // The user requirement is auto-refresh. Usually you show some indication or just update data.
    // Existing loading state 'fileLoading' will show a spinner, which might be annoying every 3 seconds.
    // But for now, let's use the existing loading state to be consistent.)
    setFileLoading(true);
    fetch(
      `${API_BASE}/api/v1/triple-diff?group=${encodeURIComponent(
        selectedGroup,
      )}&host=${encodeURIComponent(selectedHost)}&path=${encodeURIComponent(
        selectedFileKey,
      )}&mode=${isPreviewMode ? "preview" : "prod"}`,
    )
      .then((res) => {
        if (!res.ok) throw new Error("failed to fetch file diff");
        return res.json();
      })
      .then((data: TripleDiffResponse) => {
        setFileDiff(data);
      })
      .catch((err) => {
        console.error(err);
        // Suppress error on auto-refresh or keep it? Keep it for now.
      })
      .finally(() => setFileLoading(false));
  }, [refreshTrigger]);

  const menuItems: MenuProps["items"] = useMemo(() => {
    return groups.map((g) => ({
      key: g.name,
      label: g.name,
      children: g.hosts.map((h) => ({
        key: `${g.name}/${h.host}`,
        label: h.host,
      })),
    }));
  }, [groups]);

  const onMenuSelect: MenuProps["onSelect"] = (info) => {
    const [group, host] = info.key.split("/");
    const targetGroup = groups.find((g) => g.name === group);
    const targetHost = targetGroup?.hosts.find((h) => h.host === host);
    setSelectedGroup(group);
    setSelectedHost(host);
    setSelectedConfigSuffix(targetHost?.config_dir_suffix);
  };

  const handleCheckConfig = () => {
    if (!selectedHost || !selectedGroup) {
      antMessage.error("请先选择一个主机");
      return;
    }
    
    setCheckLoading(true);
    fetch(`${API_BASE}/api/v1/check?mode=${isPreviewMode ? "preview" : "prod"}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ server: selectedHost, group: selectedGroup }),
    })
      .then((res) => res.json())
      .then((data: CheckResult) => {
        setCheckResult(data);
        setCheckModalOpen(true);
      })
      .catch((err) => {
        console.error(err);
        antMessage.error("配置检查失败");
      })
      .finally(() => setCheckLoading(false));
  };

  const handleUpdatePrepare = () => {
    if (!selectedHost || !selectedGroup) {
      antMessage.error("请先选择一个主机");
      return;
    }
    if (isPreviewMode) {
      antMessage.error("更新配置仅允许在生产模式执行");
      return;
    }

    setUpdateLoading(true);
    setUpdateStage("prepare");
    fetch(`${API_BASE}/api/v1/update/prepare?mode=prod`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ server: selectedHost, group: selectedGroup }),
    })
      .then(async (res) => {
        const data = (await res.json()) as UpdatePrepareResponse;
        setUpdateResult(data);
        setUpdateModalOpen(true);
        if (!res.ok) {
          throw new Error("failed to prepare update");
        }
      })
      .catch((err) => {
        console.error(err);
        antMessage.error("更新准备失败");
      })
      .finally(() => setUpdateLoading(false));
  };

  const handleUpdateApply = () => {
    if (!selectedHost || !selectedGroup) {
      antMessage.error("请先选择一个主机");
      return;
    }
    setUpdateLoading(true);
    setUpdateStage("apply");
    fetch(`${API_BASE}/api/v1/update/apply?mode=prod`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ server: selectedHost, group: selectedGroup }),
    })
      .then(async (res) => {
        const data = (await res.json()) as UpdateApplyResponse;
        setUpdateResult(data);
        setUpdateModalOpen(true);
        if (!res.ok) {
          throw new Error("failed to apply update");
        }
      })
      .catch((err) => {
        console.error(err);
        antMessage.error("更新执行失败");
      })
      .finally(() => setUpdateLoading(false));
  };

  return (
    <AntdApp>
      <CheckResultModal
        open={checkModalOpen}
        onClose={() => setCheckModalOpen(false)}
        result={checkResult}
      />
      <UpdateResultModal
        open={updateModalOpen}
        mode={updateStage}
        loading={updateLoading}
        onCancel={() => setUpdateModalOpen(false)}
        onConfirmApply={handleUpdateApply}
        data={updateResult}
      />
      <Layout style={{ minHeight: "100vh" }}>
        <Sider collapsible collapsed={collapsedNav} onCollapse={setCollapsedNav}>
          <div
            style={{
              height: 48,
              margin: 16,
              background: "rgba(255,255,255,0.2)",
              borderRadius: 8,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              color: "white",
              fontWeight: "bold",
              overflow: "hidden",
              whiteSpace: "nowrap",
            }}
          >
            {collapsedNav ? "Repo" : "Gitops Nginx"}
          </div>
          <Menu
            mode="inline"
            theme="dark"
            items={menuItems}
            onSelect={onMenuSelect}
            selectedKeys={
              selectedGroup && selectedHost
                ? [`${selectedGroup}/${selectedHost}`]
                : []
            }
          />
        </Sider>
        <Layout>
          <Header
            style={{
              background: "#fff",
              padding: "0 16px",
              borderBottom: "1px solid #f0f0f0",
              display: "flex",
              alignItems: "center",
              justifyContent: "flex-end",
            }}
          >
            <Space>
              <GitStatusDrawer />
              <Text>生产环境</Text>
              <Switch
                checked={isPreviewMode}
                onChange={setIsPreviewMode}
                checkedChildren="预览"
                unCheckedChildren="生产"
              />
              <Text>预览环境</Text>
              <AutoRefresh onTrigger={handleRefresh} />
            </Space>
          </Header>
          <Layout style={{ padding: 16, gap: 16 }}>
            <Sider width={280} style={{ background: "#fff", padding: 12 }}>
              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "center",
                  marginBottom: 8,
                }}
              >
                <Title level={5} style={{ margin: 0 }}>
                  文件树
                </Title>
                <Switch
                  size="small"
                  checked={showAllFiles}
                  onChange={setShowAllFiles}
                  checkedChildren="全部"
                  unCheckedChildren="变更"
                />
              </div>
              <Spin spinning={treeLoading}>
                {treeData.length === 0 ? (
                  <Empty description="暂无文件" />
                ) : (
                  <Tree
                    key={treeKey}
                    treeData={treeData}
                    onSelect={handleSelectFile}
                    selectedKeys={selectedFileKey ? [selectedFileKey] : []}
                    defaultExpandAll
                    showLine
                  />
                )}
              </Spin>
              <div style={{ marginTop: 12, borderTop: "1px solid #f0f0f0", paddingTop: 8 }}>
                <Text type="secondary" style={{ fontSize: 12 }}>图例：</Text>
                <div style={{ display: "flex", flexDirection: "column", gap: 4, marginTop: 4 }}>
                  {Object.entries(STATUS_MARKERS).map(([key, { icon, color, label }]) => (
                    <div key={key} style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 12 }}>
                      <span style={{ color, fontWeight: "bold" }}>{icon}</span>
                      <Text type="secondary">{label}</Text>
                    </div>
                  ))}
                </div>
              </div>
            </Sider>
            <Content style={{ background: "#fff", padding: 16 }}>
              <Space style={{ marginBottom: 12 }}>
                <Button 
                  type="primary" 
                  loading={checkLoading}
                  onClick={handleCheckConfig}
                >
                  执行配置检查
                </Button>
                <Button
                  type="primary"
                  danger
                  loading={updateLoading}
                  onClick={handleUpdatePrepare}
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
                  <Space
                    direction="vertical"
                    style={{ width: "100%" }}
                    size="small"
                  >
                    <Text strong>远程 Nginx vs {isPreviewMode ? "预览环境" : "生产环境"} Diff</Text>
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
                  <Space
                    direction="vertical"
                    style={{ width: "100%" }}
                    size="small"
                  >
                    <Text strong>{isPreviewMode ? "预览环境" : "生产环境"} 文件内容</Text>
                    <Spin spinning={fileLoading}>
                      {fileDiff ? (
                        <Paragraph
                          style={{ whiteSpace: "pre-wrap", fontFamily: "monospace" }}
                        >
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
                  <Space
                    direction="vertical"
                    style={{ width: "100%" }}
                    size="small"
                  >
                    <Text strong>远程 Nginx 文件内容</Text>
                    <Spin spinning={fileLoading}>
                      {fileDiff ? (
                        <Paragraph
                          style={{ whiteSpace: "pre-wrap", fontFamily: "monospace" }}
                        >
                          {fileDiff.remote_content}
                        </Paragraph>
                      ) : (
                        <Empty description="未选择文件" />
                      )}
                    </Spin>
                  </Space>
                </div>
              </div>
            </Content>
          </Layout>
          <div style={{ padding: "0 16px 16px" }}>
            <Text type="secondary">
              组: {selectedGroup || "-"} | 主机:{" "}
              {selectedHost || "-"} | 配置目录后缀:{" "}
              {selectedConfigSuffix || "-"} | 模式: {isPreviewMode ? "预览" : "生产"}
            </Text>
          </div>
        </Layout>
      </Layout>
    </AntdApp>
  );
}

export default App;
