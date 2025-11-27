/**
 * FilePreview component - Modal for previewing files
 */

import React, { useState, useEffect } from 'react';
import { Modal, Spin, Typography, Button } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import { getFileUrl, downloadFile } from '../../api';
import { getPreviewType } from '../../utils';
import './FilePreview.css';

const { Text } = Typography;

interface FilePreviewProps {
  visible: boolean;
  name: string;
  path: string;
  mimeType?: string;
  onClose: () => void;
  onDownload: () => void;
}

const FilePreview: React.FC<FilePreviewProps> = ({
  visible,
  name,
  path,
  mimeType,
  onClose,
  onDownload,
}) => {
  const [loading, setLoading] = useState(true);
  const [textContent, setTextContent] = useState<string>('');
  const [error, setError] = useState<string>('');
  
  const previewType = getPreviewType(name, mimeType);
  const fileUrl = getFileUrl(path);

  // Load text content when modal opens for text files
  useEffect(() => {
    if (!visible || previewType !== 'text') {
      return;
    }
    
    let cancelled = false;
    
    // Async load function
    const loadText = async () => {
      try {
        const blob = await downloadFile(path);
        if (cancelled) return;
        const text = await blob.text();
        setTextContent(text);
        setLoading(false);
      } catch (err: unknown) {
        if (cancelled) return;
        const errorMessage = err instanceof Error ? err.message : '加载失败';
        setError(errorMessage);
        setLoading(false);
      }
    };
    
    loadText();
    
    return () => {
      cancelled = true;
    };
  }, [visible, path, previewType]);

  const renderContent = () => {
    if (!previewType) {
      return (
        <div className="preview-unsupported">
          <Text type="secondary">该文件类型不支持预览</Text>
        </div>
      );
    }

    if (loading && previewType === 'text') {
      return (
        <div className="preview-loading">
          <Spin size="large" />
        </div>
      );
    }

    if (error) {
      return (
        <div className="preview-error">
          <Text type="danger">{error}</Text>
        </div>
      );
    }

    switch (previewType) {
      case 'image':
        return (
          <img 
            src={fileUrl} 
            alt={name} 
            className="preview-image"
            onLoad={() => setLoading(false)}
          />
        );
      case 'video':
        return (
          <video 
            src={fileUrl} 
            controls 
            autoPlay 
            className="preview-video"
            onLoadedData={() => setLoading(false)}
          />
        );
      case 'audio':
        return (
          <audio 
            src={fileUrl} 
            controls 
            autoPlay 
            className="preview-audio"
          />
        );
      case 'pdf':
        return (
          <iframe 
            src={fileUrl} 
            title={name}
            className="preview-pdf"
          />
        );
      case 'text':
        return (
          <pre className="preview-text">{textContent}</pre>
        );
      default:
        return null;
    }
  };

  return (
    <Modal
      title={name}
      open={visible}
      onCancel={onClose}
      width="80%"
      style={{ top: 20 }}
      footer={[
        <Button key="download" icon={<DownloadOutlined />} onClick={onDownload}>
          下载
        </Button>,
        <Button key="close" onClick={onClose}>
          关闭
        </Button>,
      ]}
      className="preview-modal"
    >
      <div className="preview-content">
        {renderContent()}
      </div>
    </Modal>
  );
};

export default FilePreview;
