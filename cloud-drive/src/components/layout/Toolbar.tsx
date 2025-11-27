/**
 * Toolbar component with breadcrumb and actions
 */

import React from 'react';
import { Breadcrumb, Button, Space, Upload } from 'antd';
import {
  HomeOutlined,
  UploadOutlined,
  FolderAddOutlined,
} from '@ant-design/icons';
import { useAppStore } from '../../store';
import './Toolbar.css';

interface ToolbarProps {
  onNavigate?: (path: string) => void;
  onUpload?: (files: File[]) => void;
  onNewFolder?: () => void;
}

const Toolbar: React.FC<ToolbarProps> = ({ onNavigate, onUpload, onNewFolder }) => {
  const currentPath = useAppStore(state => state.currentPath);
  
  // Build breadcrumb items
  const parts = currentPath.split('/').filter(p => p);
  const breadcrumbItems = [
    {
      key: '/',
      title: (
        <span onClick={() => onNavigate?.('/')} style={{ cursor: 'pointer' }}>
          <HomeOutlined />
        </span>
      ),
    },
    ...parts.map((part, index) => {
      const path = '/' + parts.slice(0, index + 1).join('/');
      const isLast = index === parts.length - 1;
      return {
        key: path,
        title: isLast ? (
          <span>{part}</span>
        ) : (
          <span onClick={() => onNavigate?.(path)} style={{ cursor: 'pointer' }}>
            {part}
          </span>
        ),
      };
    }),
  ];

  const handleUpload = (info: { file: File }) => {
    onUpload?.([info.file]);
    return false; // Prevent default upload behavior
  };

  return (
    <div className="toolbar">
      <Breadcrumb items={breadcrumbItems} className="breadcrumb" />
      
      <Space className="toolbar-actions">
        <Upload
          beforeUpload={handleUpload as never}
          showUploadList={false}
          multiple
        >
          <Button type="primary" icon={<UploadOutlined />}>
            上传
          </Button>
        </Upload>
        <Button icon={<FolderAddOutlined />} onClick={onNewFolder}>
          新建文件夹
        </Button>
      </Space>
    </div>
  );
};

export default Toolbar;
