import type { TreeDataNode } from "antd";
import { STATUS_MARKERS } from "../types";

export function buildTree(
  prefix: string,
  paths: string[],
  fileStatuses: Record<string, string>,
  showAll: boolean
): TreeDataNode[] {
  const root: Record<string, any> = {};

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

  const process = (
    nodesMap: Record<string, any>
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

      const nodeHasChange = !!node.status || childHasChange;

      if (nodeHasChange) {
        groupHasChange = true;
      }

      if (!showAll && !nodeHasChange) {
        return;
      }

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
        _isModified: nodeHasChange,
        _rawTitle: node.rawTitle,
      });
    });

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
