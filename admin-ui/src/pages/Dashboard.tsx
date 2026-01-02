import React, { useEffect, useState } from 'react';
import { Card, Col, Row, Statistic, Spin, Alert, Typography } from 'antd';
import { CheckCircleOutlined, ClockCircleOutlined, AppstoreOutlined, HddOutlined } from '@ant-design/icons';
import { client } from '../api/client';

const { Title } = Typography;

interface HealthInfo {
  status: string;
  uptime: string;
  components: {
    database: { status: string };
    system: {
      details: {
        goroutines: number;
        alloc_mb: number;
      }
    }
  }
}

const Dashboard: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [health, setHealth] = useState<HealthInfo | null>(null);

  useEffect(() => {
    const fetchHealth = async () => {
      try {
        const res = await client.get('/health');
        setHealth(res.data);
      } catch (error) {
        console.error(error);
      } finally {
        setLoading(false);
      }
    };
    fetchHealth();
    const interval = setInterval(fetchHealth, 5000);
    return () => clearInterval(interval);
  }, []);

  const formatUptime = (uptime: string | undefined) => {
    if (!uptime) return '-';
    // Remove fractional seconds (e.g., "3m35.242273945s" -> "3m35s")
    return uptime.replace(/(\.\d+)/, '');
  };

  if (loading && !health) return <div style={{ display: 'flex', justifyContent: 'center', padding: 50 }}><Spin size="large" tip="加载系统状态..." /></div>;

  return (
    <div>
      <Title level={3} style={{ marginBottom: 24 }}>系统仪表盘</Title>
      
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={6}>
          <Card bordered={false} style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
            <Statistic 
              title="系统状态" 
              value={health?.status === 'healthy' ? '运行正常' : '异常'} 
              valueStyle={{ color: health?.status === 'healthy' ? '#3f8600' : '#cf1322', fontWeight: 'bold' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card bordered={false} style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
            <Statistic 
              title="运行时间" 
              value={formatUptime(health?.uptime)} 
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card bordered={false} style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
            <Statistic 
              title="并发协程 (Goroutines)" 
              value={health?.components?.system?.details?.goroutines || 0} 
              prefix={<AppstoreOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card bordered={false} style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
            <Statistic 
              title="内存占用 (MB)" 
              value={health?.components?.system?.details?.alloc_mb || 0} 
              prefix={<HddOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <div style={{ marginTop: 24 }}>
        <Alert
          message="系统架构说明"
          description="本系统运行在分布式模式下。大文件会自动分块并分散存储在多个 OneDrive 账号中，以突破单文件大小限制并提高传输速度。请确保配置足够的 OneDrive 账号以获得最佳性能。"
          type="info"
          showIcon
          style={{ border: '1px solid #91caff', background: '#e6f7ff' }}
        />
      </div>
    </div>
  );
};

export default Dashboard;
