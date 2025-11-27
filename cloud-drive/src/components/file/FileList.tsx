/**
 * FileList component for list view
 */

import React from 'react';
import { Table, Checkbox, Typography, Space } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  FolderFilled,
  FileImageOutlined,
  VideoCameraOutlined,
  SoundOutlined,
  FilePdfOutlined,
  FileWordOutlined,
  FileExcelOutlined,
  FilePptOutlined,
  CodeOutlined,
  FileTextOutlined,
  FileZipOutlined,
  FileOutlined,
} from '@ant-design/icons';
import type { FileItem } from '../../types';
import { formatSize, formatDate, getFileIcon } from '../../utils';
import './FileList.css';

const { Text } = Typography;

interface FileListProps {
  items: FileItem[];
  selectedPaths: Set<string>;
  onSelect?: (path: string, selected: boolean) => void;
  onDoubleClick?: (item: FileItem) => void;
  onContextMenu?: (e: React.MouseEvent, item: FileItem) => void;
}

// Icon mapping
const iconMap: Record<string, React.ReactNode> = {
  'folder': <FolderFilled style={{ fontSize: 24, color: '#fcd147' }} />,
  'file-image': <FileImageOutlined style={{ fontSize: 24, color: '#b4a0ff' }} />,
  'video-camera': <VideoCameraOutlined style={{ fontSize: 24, color: '#ff8c8c' }} />,
  'sound': <SoundOutlined style={{ fontSize: 24, color: '#8cffb4' }} />,
  'file-pdf': <FilePdfOutlined style={{ fontSize: 24, color: '#ff6b6b' }} />,
  'file-word': <FileWordOutlined style={{ fontSize: 24, color: '#2b579a' }} />,
  'file-excel': <FileExcelOutlined style={{ fontSize: 24, color: '#217346' }} />,
  'file-ppt': <FilePptOutlined style={{ fontSize: 24, color: '#d24726' }} />,
  'code': <CodeOutlined style={{ fontSize: 24, color: '#6bafff' }} />,
  'file-text': <FileTextOutlined style={{ fontSize: 24, color: '#a0a0a0' }} />,
  'file-zip': <FileZipOutlined style={{ fontSize: 24, color: '#ffd76b' }} />,
  'file': <FileOutlined style={{ fontSize: 24, color: '#8c8c8c' }} />,
};

const FileList: React.FC<FileListProps> = ({
  items,
  selectedPaths,
  onSelect,
  onDoubleClick,
  onContextMenu,
}) => {
  const columns: ColumnsType<FileItem> = [
    {
      title: '',
      dataIndex: 'select',
      width: 40,
      render: (_, record) => (
        <Checkbox
          checked={selectedPaths.has(record.path)}
          onClick={(e) => {
            e.stopPropagation();
            onSelect?.(record.path, !selectedPaths.has(record.path));
          }}
        />
      ),
    },
    {
      title: '名称',
      dataIndex: 'name',
      ellipsis: true,
      render: (_, record) => {
        const iconInfo = getFileIcon(record.type, record.name, record.mime_type);
        const icon = iconMap[iconInfo.icon] || iconMap['file'];
        return (
          <Space>
            {icon}
            <Text>{record.name}</Text>
          </Space>
        );
      },
    },
    {
      title: '大小',
      dataIndex: 'size',
      width: 100,
      render: (size, record) => (
        <Text type="secondary">
          {record.type === 'file' ? formatSize(size || 0) : '-'}
        </Text>
      ),
    },
    {
      title: '修改时间',
      dataIndex: 'updated_at',
      width: 150,
      render: (date) => (
        <Text type="secondary">{formatDate(date)}</Text>
      ),
    },
  ];

  return (
    <Table
      className="file-list"
      columns={columns}
      dataSource={items}
      rowKey="path"
      pagination={false}
      size="middle"
      onRow={(record) => ({
        onClick: () => onSelect?.(record.path, !selectedPaths.has(record.path)),
        onDoubleClick: () => onDoubleClick?.(record),
        onContextMenu: (e) => {
          e.preventDefault();
          onContextMenu?.(e, record);
        },
        className: selectedPaths.has(record.path) ? 'selected-row' : '',
      })}
    />
  );
};

export default FileList;
