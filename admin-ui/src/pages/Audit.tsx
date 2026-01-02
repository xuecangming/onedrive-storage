import React, { useEffect, useState } from 'react';
import { Card, Button, Descriptions, Table, Tag, message, Progress, Typography, Empty } from 'antd';
import { PlayCircleOutlined, SyncOutlined } from '@ant-design/icons';
import { client } from '../api/client';

const { Title } = Typography;

interface AuditIssue {
  type: string;
  bucket: string;
  key: string;
  description: string;
}

interface AuditReport {
  id: string;
  status: string;
  start_time: string;
  end_time?: string;
  total_objects: number;
  total_chunks: number;
  checked_count: number;
  issues: AuditIssue[];
  summary?: string;
}

const Audit: React.FC = () => {
  const [report, setReport] = useState<AuditReport | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchStatus = async () => {
    try {
      const res = await client.get('/audit/status');
      setReport(res.data);
    } catch (error) {
      // Ignore 404
    }
  };

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  const startAudit = async () => {
    setLoading(true);
    try {
      await client.post('/audit/start');
      message.success('审计任务已启动');
      fetchStatus();
    } catch (error) {
      message.error('启动审计失败');
    } finally {
      setLoading(false);
    }
  };

  const columns = [
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => <Tag color="red">{type.toUpperCase()}</Tag>,
    },
    {
      title: '存储桶',
      dataIndex: 'bucket',
      key: 'bucket',
    },
    {
      title: '对象 Key',
      dataIndex: 'key',
      key: 'key',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
  ];

  const percent = report ? Math.round((report.checked_count / (report.total_objects + report.total_chunks || 1)) * 100) : 0;

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={3} style={{ margin: 0 }}>审计与健康检查</Title>
        <Button 
          type="primary" 
          icon={report?.status === 'running' ? <SyncOutlined spin /> : <PlayCircleOutlined />} 
          onClick={startAudit} 
          loading={loading || report?.status === 'running'}
        >
          {report?.status === 'running' ? '审计进行中...' : '开始新审计'}
        </Button>
      </div>

      {report ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
          <Card title="当前状态" bordered={false} style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
            <div style={{ marginBottom: 24 }}>
              <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between' }}>
                <span>进度: {report.status === 'running' ? '进行中' : (report.status === 'completed' ? '已完成' : '失败')}</span>
                <span>{percent}%</span>
              </div>
              <Progress percent={percent} status={report.status === 'running' ? 'active' : (report.issues.length > 0 ? 'exception' : 'success')} />
            </div>
            
            <Descriptions bordered column={{ xxl: 4, xl: 3, lg: 3, md: 3, sm: 2, xs: 1 }}>
              <Descriptions.Item label="审计 ID">{report.id}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={report.status === 'running' ? 'blue' : (report.status === 'completed' ? 'green' : 'red')}>
                  {report.status.toUpperCase()}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="开始时间">{new Date(report.start_time).toLocaleString('zh-CN')}</Descriptions.Item>
              <Descriptions.Item label="已检查对象">{report.checked_count} / {report.total_objects + report.total_chunks}</Descriptions.Item>
              <Descriptions.Item label="发现问题" span={2}>
                <span style={{ color: report.issues.length > 0 ? 'red' : 'green', fontWeight: 'bold' }}>
                  {report.issues.length}
                </span>
              </Descriptions.Item>
              {report.summary && <Descriptions.Item label="摘要" span={3}>{report.summary}</Descriptions.Item>}
            </Descriptions>
          </Card>

          {report.issues.length > 0 && (
            <Card title="问题列表" bordered={false} style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
              <Table columns={columns} dataSource={report.issues} rowKey={(r) => r.key + r.type} pagination={{ pageSize: 5 }} />
            </Card>
          )}
        </div>
      ) : (
        <Card bordered={false} style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="暂无审计报告"
          >
            <Button type="primary" onClick={startAudit}>开始首次审计</Button>
          </Empty>
        </Card>
      )}
    </div>
  );
};

export default Audit;
