/**
 * FileGrid component - Displays files in grid layout
 */

import React from 'react';
import { Row, Col, Empty, Spin } from 'antd';
import { FolderOpenOutlined } from '@ant-design/icons';
import FileCard from './FileCard';
import type { FileItem } from '../../types';
import './FileGrid.css';

interface FileGridProps {
  items: FileItem[];
  loading?: boolean;
  selectedPaths: Set<string>;
  onSelect?: (path: string, ctrlKey: boolean, shiftKey: boolean) => void;
  onDoubleClick?: (item: FileItem) => void;
  onContextMenu?: (e: React.MouseEvent, item: FileItem) => void;
}

const FileGrid: React.FC<FileGridProps> = ({
  items,
  loading,
  selectedPaths,
  onSelect,
  onDoubleClick,
  onContextMenu,
}) => {
  if (loading) {
    return (
      <div className="file-grid-loading">
        <Spin size="large" />
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <Empty
        className="file-grid-empty"
        image={<FolderOpenOutlined style={{ fontSize: 64, color: '#d9d9d9' }} />}
        description="文件夹为空"
      />
    );
  }

  // Sort: directories first, then by name
  const sortedItems = [...items].sort((a, b) => {
    if (a.type === 'directory' && b.type !== 'directory') return -1;
    if (a.type !== 'directory' && b.type === 'directory') return 1;
    return (a.name || '').localeCompare(b.name || '');
  });

  return (
    <div className="file-grid">
      <Row gutter={[16, 16]}>
        {sortedItems.map((item) => (
          <Col key={item.path} xs={12} sm={8} md={6} lg={4} xl={3}>
            <FileCard
              item={item}
              selected={selectedPaths.has(item.path)}
              onClick={(e) => onSelect?.(item.path, e.ctrlKey || e.metaKey, e.shiftKey)}
              onDoubleClick={() => onDoubleClick?.(item)}
              onContextMenu={(e) => {
                e.preventDefault();
                onContextMenu?.(e, item);
              }}
            />
          </Col>
        ))}
      </Row>
    </div>
  );
};

export default FileGrid;
