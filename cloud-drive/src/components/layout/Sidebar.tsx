/**
 * Sidebar component
 */

import React from 'react';
import { Menu, Progress, Typography, Space } from 'antd';
import {
  FolderOutlined,
  ClockCircleOutlined,
  StarOutlined,
  DeleteOutlined,
  CloudOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { useAppStore } from '../../store';
import { formatSize } from '../../utils';
import './Sidebar.css';

const { Text } = Typography;

interface SidebarProps {
  collapsed?: boolean;
  onSettingsClick?: () => void;
}

const Sidebar: React.FC<SidebarProps> = ({ collapsed, onSettingsClick }) => {
  const spaceInfo = useAppStore(state => state.spaceInfo);
  
  const menuItems = [
    {
      key: 'files',
      icon: <FolderOutlined />,
      label: '我的文件',
    },
    {
      key: 'recent',
      icon: <ClockCircleOutlined />,
      label: '最近',
      disabled: true,
    },
    {
      key: 'starred',
      icon: <StarOutlined />,
      label: '收藏',
      disabled: true,
    },
    {
      key: 'trash',
      icon: <DeleteOutlined />,
      label: '回收站',
      disabled: true,
    },
  ];

  const usagePercent = spaceInfo ? (spaceInfo.used_space / spaceInfo.total_space) * 100 : 0;

  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <CloudOutlined className="sidebar-logo-icon" />
        {!collapsed && <span className="sidebar-logo-text">云盘</span>}
      </div>
      
      <Menu
        mode="inline"
        defaultSelectedKeys={['files']}
        items={menuItems}
        className="sidebar-menu"
      />
      
      {!collapsed && spaceInfo && (
        <div className="sidebar-storage">
          <Space direction="vertical" style={{ width: '100%' }}>
            <Text type="secondary">
              <CloudOutlined style={{ marginRight: 8 }} />
              存储空间
            </Text>
            <Progress 
              percent={Math.round(usagePercent)} 
              showInfo={false}
              strokeColor="#1677ff"
              size="small"
            />
            <Text type="secondary" style={{ fontSize: 12 }}>
              {formatSize(spaceInfo.used_space)} / {formatSize(spaceInfo.total_space)}
            </Text>
          </Space>
        </div>
      )}
      
      <div className="sidebar-footer">
        <Menu
          mode="inline"
          selectable={false}
          items={[
            {
              key: 'settings',
              icon: <SettingOutlined />,
              label: '设置',
              onClick: onSettingsClick,
            },
          ]}
        />
      </div>
    </div>
  );
};

export default Sidebar;
