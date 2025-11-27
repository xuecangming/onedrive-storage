/**
 * ContextMenu component for file operations
 */

import React, { useEffect, useRef } from 'react';
import { Menu } from 'antd';
import {
  FolderOpenOutlined,
  EyeOutlined,
  DownloadOutlined,
  EditOutlined,
  ScissorOutlined,
  CopyOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import type { FileItem } from '../../types';
import { getPreviewType } from '../../utils';
import './ContextMenu.css';

interface ContextMenuProps {
  visible: boolean;
  x: number;
  y: number;
  item: FileItem | null;
  onClose: () => void;
  onOpen?: () => void;
  onPreview?: () => void;
  onDownload?: () => void;
  onRename?: () => void;
  onMove?: () => void;
  onCopy?: () => void;
  onDelete?: () => void;
}

const ContextMenu: React.FC<ContextMenuProps> = ({
  visible,
  x,
  y,
  item,
  onClose,
  onOpen,
  onPreview,
  onDownload,
  onRename,
  onMove,
  onCopy,
  onDelete,
}) => {
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose();
      }
    };

    if (visible) {
      document.addEventListener('click', handleClickOutside);
    }

    return () => {
      document.removeEventListener('click', handleClickOutside);
    };
  }, [visible, onClose]);

  if (!visible || !item) return null;

  const isDirectory = item.type === 'directory';
  const canPreview = !isDirectory && getPreviewType(item.name, item.mime_type);

  const menuItems = [
    isDirectory && {
      key: 'open',
      icon: <FolderOpenOutlined />,
      label: '打开',
      onClick: () => { onOpen?.(); onClose(); },
    },
    canPreview && {
      key: 'preview',
      icon: <EyeOutlined />,
      label: '预览',
      onClick: () => { onPreview?.(); onClose(); },
    },
    !isDirectory && {
      key: 'download',
      icon: <DownloadOutlined />,
      label: '下载',
      onClick: () => { onDownload?.(); onClose(); },
    },
    { type: 'divider' },
    {
      key: 'rename',
      icon: <EditOutlined />,
      label: '重命名',
      onClick: () => { onRename?.(); onClose(); },
    },
    {
      key: 'move',
      icon: <ScissorOutlined />,
      label: '移动到',
      onClick: () => { onMove?.(); onClose(); },
    },
    !isDirectory && {
      key: 'copy',
      icon: <CopyOutlined />,
      label: '复制到',
      onClick: () => { onCopy?.(); onClose(); },
    },
    { type: 'divider' },
    {
      key: 'delete',
      icon: <DeleteOutlined />,
      label: '删除',
      danger: true,
      onClick: () => { onDelete?.(); onClose(); },
    },
  ].filter(Boolean);

  // Adjust position to stay within viewport
  const adjustedX = Math.min(x, window.innerWidth - 180);
  const adjustedY = Math.min(y, window.innerHeight - 300);

  return (
    <div
      ref={menuRef}
      className="context-menu"
      style={{ left: adjustedX, top: adjustedY }}
    >
      <Menu
        items={menuItems as never}
        mode="vertical"
        selectable={false}
      />
    </div>
  );
};

export default ContextMenu;
