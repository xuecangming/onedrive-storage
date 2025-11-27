/**
 * Upload Panel component - Shows upload progress
 */

import React from 'react';
import { Card, List, Progress, Typography, Button, Space } from 'antd';
import { CloseOutlined, FileOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { useAppStore } from '../../store';
import './UploadPanel.css';

const { Text } = Typography;

interface UploadPanelProps {
  visible: boolean;
  onClose: () => void;
}

const UploadPanel: React.FC<UploadPanelProps> = ({ visible, onClose }) => {
  const uploads = useAppStore(state => state.uploads);
  
  if (!visible || uploads.size === 0) return null;

  const uploadList = Array.from(uploads.values()).reverse();

  return (
    <Card
      className="upload-panel"
      title="上传任务"
      extra={
        <Button type="text" icon={<CloseOutlined />} onClick={onClose} size="small" />
      }
      size="small"
    >
      <List
        dataSource={uploadList}
        renderItem={(item) => (
          <List.Item className="upload-item">
            <Space direction="vertical" style={{ width: '100%' }}>
              <Space>
                {item.status === 'success' ? (
                  <CheckCircleOutlined style={{ color: '#52c41a' }} />
                ) : item.status === 'error' ? (
                  <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
                ) : (
                  <FileOutlined />
                )}
                <Text ellipsis style={{ maxWidth: 200 }}>{item.filename}</Text>
              </Space>
              
              {item.status === 'uploading' && (
                <Progress percent={item.progress} size="small" />
              )}
              
              {item.status === 'error' && (
                <Text type="danger" style={{ fontSize: 12 }}>{item.error}</Text>
              )}
              
              {item.status === 'success' && (
                <Text type="success" style={{ fontSize: 12 }}>上传完成</Text>
              )}
            </Space>
          </List.Item>
        )}
      />
    </Card>
  );
};

export default UploadPanel;
