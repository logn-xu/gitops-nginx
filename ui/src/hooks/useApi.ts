import { useCallback } from "react";
import { App } from "antd";
import { API_BASE, GroupsResponse, TreeResponse, TripleDiffResponse } from "../types";
import type { CheckResult } from "../components/CheckResultModal";
import type { UpdatePrepareResponse, UpdateApplyResponse } from "../components/UpdateResultModal";

export function useApi() {
  const { message: antMessage } = App.useApp();

  const fetchGroups = useCallback(async (): Promise<GroupsResponse | null> => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/groups`);
      return await res.json();
    } catch (err) {
      console.error(err);
      antMessage.error("获取分组列表失败");
      return null;
    }
  }, [antMessage]);

  const fetchTree = useCallback(
    async (group: string, host: string, mode: string): Promise<TreeResponse | null> => {
      try {
        const res = await fetch(
          `${API_BASE}/api/v1/tree?group=${encodeURIComponent(group)}&host=${encodeURIComponent(host)}&mode=${mode}`
        );
        return await res.json();
      } catch (err) {
        console.error(err);
        antMessage.error("获取文件树失败");
        return null;
      }
    },
    [antMessage]
  );

  const fetchFileDiff = useCallback(
    async (group: string, host: string, path: string, mode: string): Promise<TripleDiffResponse | null> => {
      try {
        const res = await fetch(
          `${API_BASE}/api/v1/triple-diff?group=${encodeURIComponent(group)}&host=${encodeURIComponent(host)}&path=${encodeURIComponent(path)}&mode=${mode}`
        );
        if (!res.ok) throw new Error("failed to fetch file diff");
        return await res.json();
      } catch (err) {
        console.error(err);
        antMessage.error("获取文件内容/差异失败");
        return null;
      }
    },
    [antMessage]
  );

  const checkConfig = useCallback(
    async (group: string, host: string, mode: string): Promise<CheckResult | null> => {
      try {
        const res = await fetch(`${API_BASE}/api/v1/check?mode=${mode}`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ server: host, group }),
        });
        return await res.json();
      } catch (err) {
        console.error(err);
        antMessage.error("配置检查失败");
        return null;
      }
    },
    [antMessage]
  );

  const updatePrepare = useCallback(
    async (group: string, host: string): Promise<UpdatePrepareResponse | null> => {
      try {
        const res = await fetch(`${API_BASE}/api/v1/update/prepare?mode=prod`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ server: host, group }),
        });
        const data = await res.json();
        if (!res.ok) throw new Error("failed to prepare update");
        return data;
      } catch (err) {
        console.error(err);
        antMessage.error("更新准备失败");
        return null;
      }
    },
    [antMessage]
  );

  const updateApply = useCallback(
    async (group: string, host: string): Promise<UpdateApplyResponse | null> => {
      try {
        const res = await fetch(`${API_BASE}/api/v1/update/apply?mode=prod`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ server: host, group }),
        });
        const data = await res.json();
        if (!res.ok) throw new Error("failed to apply update");
        return data;
      } catch (err) {
        console.error(err);
        antMessage.error("更新执行失败");
        return null;
      }
    },
    [antMessage]
  );

  return { fetchGroups, fetchTree, fetchFileDiff, checkConfig, updatePrepare, updateApply };
}
