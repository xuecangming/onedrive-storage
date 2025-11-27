/**
 * Header component with search and actions
 */

import React from 'react';
import { Input, Button, Space, Tooltip } from 'antd';
import {
  SearchOutlined,
  AppstoreOutlined,
  UnorderedListOutlined,
  ReloadOutlined,
  MenuOutlined,
} from '@ant-design/icons';
import { useAppStore } from '../../store';
import './Header.css';

interface HeaderProps {
  onSearch?: (value: string) => void;
  onRefresh?: () => void;
  onMenuClick?: () => void;
}

const Header: React.FC<HeaderProps> = ({ onSearch, onRefresh, onMenuClick }) => {
  const viewMode = useAppStore(state => state.viewMode);
  const setViewMode = useAppStore(state => state.setViewMode);

  const toggleView = () => {
    setViewMode(viewMode === 'grid' ? 'list' : 'grid');
  };

  return (
    <div className="app-header">
      <div className="header-left">
        <Button 
          type="text" 
          icon={<MenuOutlined />} 
          onClick={onMenuClick}
          className="mobile-menu-btn"
        />
        <Input
          placeholder="搜索文件..."
          prefix={<SearchOutlined />}
          onChange={(e) => onSearch?.(e.target.value)}
          className="search-input"
          allowClear
        />
      </div>
      
      <div className="header-right">
        <Space>
          <Tooltip title={viewMode === 'grid' ? '列表视图' : '网格视图'}>
            <Button 
              type="text" 
              icon={viewMode === 'grid' ? <UnorderedListOutlined /> : <AppstoreOutlined />}
              onClick={toggleView}
            />
          </Tooltip>
          <Tooltip title="刷新">
            <Button 
              type="text" 
              icon={<ReloadOutlined />}
              onClick={onRefresh}
            />
          </Tooltip>
        </Space>
      </div>
    </div>
  );
};

export default Header;
