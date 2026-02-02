export type GroupSummary = {
  name: string;
  hosts: { name: string; host: string; config_dir_suffix: string }[];
};

export type GroupsResponse = {
  groups: GroupSummary[];
};

export type TreeResponse = {
  prefix: string;
  paths: string[];
  diff_paths?: string[];
  file_statuses?: Record<string, string>;
};

export type TripleDiffResponse = {
  path: string;
  remote_content: string;
  compare_content: string;
  diff: string;
  mode: string;
  compare_label: string;
  file_status?: string;
};

export const API_BASE = import.meta.env.VITE_API_BASE || "";

export const STATUS_MARKERS: Record<string, { icon: string; color: string; label: string }> = {
  modified: { icon: "★", color: "#faad14", label: "修改" },
  added: { icon: "+", color: "#52c41a", label: "新增" },
  deleted: { icon: "-", color: "#ff4d4f", label: "删除" },
};
