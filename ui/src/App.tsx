import { Key, useEffect, useMemo, useState } from "react";
import { App as AntdApp, Layout, Menu, Space, Switch, Typography } from "antd";
import type { MenuProps, TreeDataNode } from "antd";
import { AutoRefresh } from "./components/AutoRefresh";
import { GitStatusDrawer } from "./components/GitStatusDrawer";
import { CheckResultModal, type CheckResult } from "./components/CheckResultModal";
import { UpdateResultModal, type UpdateApplyResponse, type UpdatePrepareResponse } from "./components/UpdateResultModal";
import { FileTree } from "./components/FileTree";
import { FileContentPanel } from "./components/FileContentPanel";
import { useApi } from "./hooks/useApi";
import { buildTree } from "./utils/treeBuilder";
import type { GroupSummary, TripleDiffResponse } from "./types";

const { Header, Sider, Content } = Layout;
const { Text } = Typography;

function App() {
  const { fetchGroups, fetchTree, fetchFileDiff, checkConfig, updatePrepare, updateApply } = useApi();
  const { message: antMessage } = AntdApp.useApp();

  const [collapsedNav, setCollapsedNav] = useState(false);
  const [groups, setGroups] = useState<GroupSummary[]>([]);
  const [selectedGroup, setSelectedGroup] = useState<string>();
  const [selectedHost, setSelectedHost] = useState<string>();
  const [selectedConfigSuffix, setSelectedConfigSuffix] = useState<string>();
  const [isPreviewMode, setIsPreviewMode] = useState(true);
  const [showAllFiles, setShowAllFiles] = useState(true);
  const [openKeys, setOpenKeys] = useState<string[]>([]);

  const [treeData, setTreeData] = useState<TreeDataNode[]>([]);
  const [treeKey, setTreeKey] = useState(0);
  const [treeLoading, setTreeLoading] = useState(false);

  const [selectedFileKey, setSelectedFileKey] = useState<string>();
  const [fileLoading, setFileLoading] = useState(false);
  const [fileDiff, setFileDiff] = useState<TripleDiffResponse>();
  const [checkLoading, setCheckLoading] = useState(false);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const [checkModalOpen, setCheckModalOpen] = useState(false);
  const [checkResult, setCheckResult] = useState<CheckResult>();

  const [updateLoading, setUpdateLoading] = useState(false);
  const [updateModalOpen, setUpdateModalOpen] = useState(false);
  const [updateStage, setUpdateStage] = useState<"prepare" | "apply">("prepare");
  const [updateResult, setUpdateResult] = useState<UpdatePrepareResponse | UpdateApplyResponse>();

  const mode = isPreviewMode ? "preview" : "prod";

  useEffect(() => {
    fetchGroups().then((data) => {
      if (!data) return;
      setGroups(data.groups || []);
      const firstGroup = data.groups?.[0];
      const firstHost = firstGroup?.hosts?.[0];
      if (firstGroup && firstHost) {
        setSelectedGroup(firstGroup.name);
        setSelectedHost(firstHost.host);
        setSelectedConfigSuffix(firstHost.config_dir_suffix);
        setOpenKeys([firstGroup.name]);
      }
    });
  }, [fetchGroups]);

  useEffect(() => {
    setSelectedFileKey(undefined);
    setFileDiff(undefined);
  }, [selectedGroup, selectedHost, isPreviewMode]);

  useEffect(() => {
    if (!selectedGroup || !selectedHost) return;
    setTreeLoading(true);
    fetchTree(selectedGroup, selectedHost, mode).then((data) => {
      if (data) {
        setTreeData(buildTree(data.prefix, data.paths || [], data.file_statuses || {}, showAllFiles));
        setTreeKey((k) => k + 1);
      }
      setTreeLoading(false);
    });
  }, [selectedGroup, selectedHost, mode, fetchTree, refreshTrigger, showAllFiles]);

  const loadFileDiff = (path: string) => {
    if (!selectedGroup || !selectedHost) return;
    setFileLoading(true);
    fetchFileDiff(selectedGroup, selectedHost, path, mode).then((data) => {
      if (data) setFileDiff(data);
      setFileLoading(false);
    });
  };

  const handleSelectFile = (keys: Key[]) => {
    const key = keys[0] as string | undefined;
    if (!key) return;
    setSelectedFileKey(key);
    loadFileDiff(key);
  };

  useEffect(() => {
    if (selectedFileKey) loadFileDiff(selectedFileKey);
  }, [refreshTrigger]);

  const menuItems: MenuProps["items"] = useMemo(() => {
    return groups.map((g) => ({
      key: g.name,
      label: g.name,
      children: g.hosts.map((h) => ({ key: `${g.name}/${h.host}`, label: h.host })),
    }));
  }, [groups]);

  const onMenuSelect: MenuProps["onSelect"] = (info) => {
    const [group, host] = info.key.split("/");
    const targetHost = groups.find((g) => g.name === group)?.hosts.find((h) => h.host === host);
    setSelectedGroup(group);
    setSelectedHost(host);
    setSelectedConfigSuffix(targetHost?.config_dir_suffix);
  };

  const handleCheckConfig = async () => {
    if (!selectedHost || !selectedGroup) {
      antMessage.error("请先选择一个主机");
      return;
    }
    setCheckLoading(true);
    const result = await checkConfig(selectedGroup, selectedHost, mode);
    if (result) {
      setCheckResult(result);
      setCheckModalOpen(true);
    }
    setCheckLoading(false);
  };

  const handleUpdatePrepare = async () => {
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
    const result = await updatePrepare(selectedGroup, selectedHost);
    if (result) {
      setUpdateResult(result);
      setUpdateModalOpen(true);
    }
    setUpdateLoading(false);
  };

  const handleUpdateApply = async () => {
    if (!selectedHost || !selectedGroup) {
      antMessage.error("请先选择一个主机");
      return;
    }
    setUpdateLoading(true);
    setUpdateStage("apply");
    const result = await updateApply(selectedGroup, selectedHost);
    if (result) {
      setUpdateResult(result);
      setUpdateModalOpen(true);
    }
    setUpdateLoading(false);
  };

  return (
    <AntdApp>
      <CheckResultModal open={checkModalOpen} onClose={() => setCheckModalOpen(false)} result={checkResult} />
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
            openKeys={openKeys}
            onOpenChange={setOpenKeys}
            selectedKeys={selectedGroup && selectedHost ? [`${selectedGroup}/${selectedHost}`] : []}
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
              <Switch checked={isPreviewMode} onChange={setIsPreviewMode} checkedChildren="预览" unCheckedChildren="生产" />
              <Text>预览环境</Text>
              <AutoRefresh onTrigger={() => setRefreshTrigger((p) => p + 1)} />
            </Space>
          </Header>
          <Layout style={{ padding: 16, gap: 16 }}>
            <Sider width={280} style={{ background: "#fff", padding: 12 }}>
              <FileTree
                treeData={treeData}
                treeKey={treeKey}
                loading={treeLoading}
                selectedFileKey={selectedFileKey}
                showAllFiles={showAllFiles}
                onShowAllFilesChange={setShowAllFiles}
                onSelectFile={handleSelectFile}
              />
            </Sider>
            <Content style={{ background: "#fff", padding: 16 }}>
              <FileContentPanel
                selectedFileKey={selectedFileKey}
                fileDiff={fileDiff}
                fileLoading={fileLoading}
                isPreviewMode={isPreviewMode}
                checkLoading={checkLoading}
                updateLoading={updateLoading}
                onCheckConfig={handleCheckConfig}
                onUpdatePrepare={handleUpdatePrepare}
              />
            </Content>
          </Layout>
          <div style={{ padding: "0 16px 16px" }}>
            <Text type="secondary">
              组: {selectedGroup || "-"} | 主机: {selectedHost || "-"} | 配置目录后缀: {selectedConfigSuffix || "-"} | 模式:{" "}
              {isPreviewMode ? "预览" : "生产"}
            </Text>
          </div>
        </Layout>
      </Layout>
    </AntdApp>
  );
}

export default App;
