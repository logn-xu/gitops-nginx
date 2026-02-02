import { Key } from "react";
import { Empty, Spin, Switch, Tree, Typography } from "antd";
import type { TreeDataNode } from "antd";
import { STATUS_MARKERS } from "../types";

const { Title, Text } = Typography;

interface FileTreeProps {
  treeData: TreeDataNode[];
  treeKey: number;
  loading: boolean;
  selectedFileKey?: string;
  showAllFiles: boolean;
  onShowAllFilesChange: (checked: boolean) => void;
  onSelectFile: (keys: Key[]) => void;
}

export function FileTree({
  treeData,
  treeKey,
  loading,
  selectedFileKey,
  showAllFiles,
  onShowAllFilesChange,
  onSelectFile,
}: FileTreeProps) {
  return (
    <>
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
          onChange={onShowAllFilesChange}
          checkedChildren="全部"
          unCheckedChildren="变更"
        />
      </div>
      <Spin spinning={loading}>
        {treeData.length === 0 ? (
          <Empty description="暂无文件" />
        ) : (
          <Tree
            key={treeKey}
            treeData={treeData}
            onSelect={onSelectFile}
            selectedKeys={selectedFileKey ? [selectedFileKey] : []}
            defaultExpandAll
            showLine
          />
        )}
      </Spin>
      <div style={{ marginTop: 12, borderTop: "1px solid #f0f0f0", paddingTop: 8 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>
          图例：
        </Text>
        <div style={{ display: "flex", flexDirection: "column", gap: 4, marginTop: 4 }}>
          {Object.entries(STATUS_MARKERS).map(([key, { icon, color, label }]) => (
            <div key={key} style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 12 }}>
              <span style={{ color, fontWeight: "bold" }}>{icon}</span>
              <Text type="secondary">{label}</Text>
            </div>
          ))}
        </div>
      </div>
    </>
  );
}
