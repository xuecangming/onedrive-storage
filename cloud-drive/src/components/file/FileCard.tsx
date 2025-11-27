/**
 * FileCard component for grid view
 */

import React from 'react';
import { Card, Typography, Checkbox, Tooltip } from 'antd';
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
import './FileCard.css';

const { Text } = Typography;

interface FileCardProps {
  item: FileItem;
  selected?: boolean;
  onClick?: (e: React.MouseEvent) => void;
  onDoubleClick?: () => void;
  onContextMenu?: (e: React.MouseEvent) => void;
}

// Icon mapping
const iconMap: Record<string, React.ReactNode> = {
  'folder': <FolderFilled style={{ fontSize: 48, color: '#fcd147' }} />,
  'file-image': <FileImageOutlined style={{ fontSize: 48, color: '#b4a0ff' }} />,
  'video-camera': <VideoCameraOutlined style={{ fontSize: 48, color: '#ff8c8c' }} />,
  'sound': <SoundOutlined style={{ fontSize: 48, color: '#8cffb4' }} />,
  'file-pdf': <FilePdfOutlined style={{ fontSize: 48, color: '#ff6b6b' }} />,
  'file-word': <FileWordOutlined style={{ fontSize: 48, color: '#2b579a' }} />,
  'file-excel': <FileExcelOutlined style={{ fontSize: 48, color: '#217346' }} />,
  'file-ppt': <FilePptOutlined style={{ fontSize: 48, color: '#d24726' }} />,
  'code': <CodeOutlined style={{ fontSize: 48, color: '#6bafff' }} />,
  'file-text': <FileTextOutlined style={{ fontSize: 48, color: '#a0a0a0' }} />,
  'file-zip': <FileZipOutlined style={{ fontSize: 48, color: '#ffd76b' }} />,
  'file': <FileOutlined style={{ fontSize: 48, color: '#8c8c8c' }} />,
};

const FileCard: React.FC<FileCardProps> = ({
  item,
  selected,
  onClick,
  onDoubleClick,
  onContextMenu,
}) => {
  const iconInfo = getFileIcon(item.type, item.name, item.mime_type);
  const icon = iconMap[iconInfo.icon] || iconMap['file'];
  
  return (
    <Card
      className={`file-card ${selected ? 'selected' : ''}`}
      onClick={onClick}
      onDoubleClick={onDoubleClick}
      onContextMenu={onContextMenu}
      hoverable
    >
      <div className="file-card-checkbox">
        <Checkbox checked={selected} onClick={(e) => e.stopPropagation()} />
      </div>
      
      <div className="file-card-icon">
        {icon}
      </div>
      
      <Tooltip title={item.name}>
        <Text className="file-card-name" ellipsis>
          {item.name}
        </Text>
      </Tooltip>
      
      <Text className="file-card-meta" type="secondary">
        {item.type === 'file' ? formatSize(item.size || 0) : formatDate(item.created_at)}
      </Text>
    </Card>
  );
};

export default FileCard;
