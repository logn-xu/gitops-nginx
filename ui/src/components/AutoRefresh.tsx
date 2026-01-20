import React, { useState, useEffect } from "react";
import { Space, Switch, InputNumber, Typography } from "antd";

const { Text } = Typography;

interface AutoRefreshProps {
  onTrigger: () => void;
}

export const AutoRefresh: React.FC<AutoRefreshProps> = ({ onTrigger }) => {
  const [enabled, setEnabled] = useState(false);
  const [intervalSeconds, setIntervalSeconds] = useState(5);

  useEffect(() => {
    if (!enabled) return;

    // Initial trigger when enabled? Maybe not, just wait for interval.
    
    const timer = setInterval(() => {
      onTrigger();
    }, intervalSeconds * 1000);

    return () => clearInterval(timer);
  }, [enabled, intervalSeconds, onTrigger]);

  return (
    <Space style={{ marginLeft: 16, borderLeft: "1px solid #f0f0f0", paddingLeft: 16 }}>
      <Text>自动刷新</Text>
      <Switch 
        checked={enabled} 
        onChange={setEnabled} 
        size="small" 
      />
      {enabled && (
        <InputNumber
          min={3}
          value={intervalSeconds}
          onChange={(val) => setIntervalSeconds(val || 3)}
          size="small"
          style={{ width: 70 }}
          formatter={(value) => `${value}s`}
          parser={(value) => value?.replace('s', '') as unknown as number}
        />
      )}
    </Space>
  );
};
