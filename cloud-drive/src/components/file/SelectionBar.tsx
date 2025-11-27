/**
 * Selection Bar component - Shows when items are selected
 */

import React from 'react';
import { Space, Button, Typography } from 'antd';
import {
  DownloadOutlined,
  ScissorOutlined,
  CopyOutlined,
  DeleteOutlined,
  CloseOutlined,
} from '@ant-design/icons';
import './SelectionBar.css';

const { Text } = Typography;

interface SelectionBarProps {
  count: number;
  onDownload?: () => void;
  onMove?: () => void;
  onCopy?: () => void;
  onDelete?: () => void;
  onCancel?: () => void;
}

const SelectionBar: React.FC<SelectionBarProps> = ({
  count,
  onDownload,
  onMove,
  onCopy,
  onDelete,
  onCancel,
}) => {
  if (count === 0) return null;

  return (
    <div className="selection-bar">
      <Text className="selection-count">{count} 个项目已选中</Text>
      
      <Space className="selection-actions">
        <Button type="text" icon={<DownloadOutlined />} onClick={onDownload}>
          下载
        </Button>
        <Button type="text" icon={<ScissorOutlined />} onClick={onMove}>
          移动
        </Button>
        <Button type="text" icon={<CopyOutlined />} onClick={onCopy}>
          复制
        </Button>
        <Button type="text" icon={<DeleteOutlined />} danger onClick={onDelete}>
          删除
        </Button>
        <Button type="text" icon={<CloseOutlined />} onClick={onCancel}>
          取消
        </Button>
      </Space>
    </div>
  );
};

export default SelectionBar;
