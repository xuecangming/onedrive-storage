import React, { useEffect, useState } from 'react';
import { Table, Button, message, Tag, Space, Card, Typography, Drawer, Steps, Form, Input, Divider, Row, Col, Popconfirm } from 'antd';
import { SyncOutlined, PlusOutlined, UserOutlined, KeyOutlined, GlobalOutlined, DeleteOutlined } from '@ant-design/icons';
import { client } from '../api/client';

const { Title, Paragraph, Link } = Typography;

interface Account {
  id: string;
  name: string;
  email: string;
  total_space: number;
  used_space: number;
  status: string;
  last_sync: string;
}

const Accounts: React.FC = () => {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [form] = Form.useForm();

  const fetchAccounts = async () => {
    setLoading(true);
    try {
      const res = await client.get('/accounts');
      setAccounts(res.data.accounts || []);
    } catch (error) {
      message.error('获取账号列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAccounts();
  }, []);

  const handleSync = async (id: string) => {
    try {
      await client.post(`/accounts/${id}/sync`);
      message.success('同步任务已启动');
      fetchAccounts();
    } catch (error) {
      message.error('同步失败');
    }
  };

  const handleAddAccount = async (values: any) => {
    try {
      // 1. Create account via REST API
      const res = await client.post('/accounts', {
        name: values.name,
        email: values.email, // Optional in backend but good to have
        client_id: values.client_id,
        client_secret: values.client_secret,
        tenant_id: values.tenant_id || 'common',
        status: 'pending',
        priority: 10
      });
      
      const newAccount = res.data;
      
      // 2. Redirect to authorization
      message.loading('正在跳转至 Microsoft 登录页面...', 2);
      setTimeout(() => {
        window.location.href = `/api/v1/oauth/authorize/${newAccount.id}`;
      }, 1000);
      
    } catch (error) {
      message.error('创建账号失败');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await client.delete(`/accounts/${id}`);
      message.success('账号已删除');
      fetchAccounts();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <span style={{ fontWeight: 500 }}>{text}</span>,
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        let color = 'default';
        let text = '未知';
        switch (status) {
          case 'active': color = 'success'; text = '正常'; break;
          case 'pending': color = 'warning'; text = '待授权'; break;
          case 'error': color = 'error'; text = '异常'; break;
        }
        return <Tag color={color}>{text}</Tag>;
      },
    },
    {
      title: '空间使用',
      key: 'space',
      render: (_: any, record: Account) => {
        if (!record.total_space) return '-';
        const percent = Math.round((record.used_space / record.total_space) * 100);
        const usedGB = (record.used_space / 1024 / 1024 / 1024).toFixed(2);
        const totalGB = (record.total_space / 1024 / 1024 / 1024).toFixed(2);
        return (
          <div>
            <div style={{ fontSize: 12, color: '#666' }}>{usedGB} GB / {totalGB} GB</div>
            <div style={{ height: 6, background: '#f0f0f0', borderRadius: 3, marginTop: 4, overflow: 'hidden' }}>
              <div style={{ width: `${percent}%`, background: percent > 90 ? '#ff4d4f' : '#1890ff', height: '100%' }} />
            </div>
          </div>
        );
      },
    },
    {
      title: '上次同步',
      dataIndex: 'last_sync',
      key: 'last_sync',
      render: (date: string) => date ? new Date(date).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Account) => (
        <Space size="middle">
          <Button 
            type="link" 
            icon={<SyncOutlined />} 
            onClick={() => handleSync(record.id)}
            disabled={record.status !== 'active'}
          >
            同步
          </Button>
          {record.status === 'pending' && (
             <Button type="link" href={`/api/v1/oauth/authorize/${record.id}`}>
               去授权
             </Button>
          )}
          <Popconfirm
            title="确定要删除这个账号吗？"
            description="删除后将无法访问该账号下的文件"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={3} style={{ margin: 0 }}>账号管理</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setDrawerOpen(true)}>
          添加账号
        </Button>
      </div>

      <Card bordered={false} style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
        <Table 
          columns={columns} 
          dataSource={accounts} 
          rowKey="id" 
          loading={loading} 
          pagination={{ pageSize: 10 }}
          locale={{ emptyText: '暂无账号' }}
        />
      </Card>

      <Drawer
        title="添加 OneDrive 账号"
        width={720}
        onClose={() => setDrawerOpen(false)}
        open={drawerOpen}
        bodyStyle={{ paddingBottom: 80 }}
      >
        <Steps
          direction="vertical"
          current={-1}
          items={[
            {
              title: '注册应用',
              description: (
                <div>
                  <Paragraph>
                    访问 <Link href="https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationsListBlade" target="_blank">Azure Portal</Link> 并注册一个新应用。
                  </Paragraph>
                  <Paragraph>
                    <ul>
                      <li>账户类型选择: <strong>任何组织目录(任何 Microsoft Entra ID 租户 - 多租户)和个人 Microsoft 账户</strong></li>
                      <li>重定向 URI (Web): <code>{window.location.protocol}//{window.location.host}/api/v1/oauth/callback</code></li>
                    </ul>
                  </Paragraph>
                </div>
              ),
            },
            {
              title: '获取凭证',
              description: (
                <div>
                  <Paragraph>
                    在应用概览页面复制 <strong>应用程序(客户端) ID</strong>。
                  </Paragraph>
                  <Paragraph>
                    在"证书和密码"页面创建一个新的客户端密码，并复制 <strong>值</strong> (不是 Secret ID)。
                  </Paragraph>
                </div>
              ),
            },
            {
              title: '填写信息',
              description: '将获取到的信息填入下方表单。',
            }
          ]}
        />
        
        <Divider />

        <Form layout="vertical" form={form} onFinish={handleAddAccount} requiredMark="optional">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="账号备注名"
                rules={[{ required: true, message: '请输入账号备注名' }]}
              >
                <Input placeholder="例如: Personal OneDrive" prefix={<UserOutlined />} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="email"
                label="邮箱 (可选)"
              >
                <Input placeholder="user@example.com" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={24}>
              <Form.Item
                name="client_id"
                label="应用程序(客户端) ID"
                rules={[{ required: true, message: '请输入 Client ID' }]}
              >
                <Input placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" prefix={<GlobalOutlined />} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={24}>
              <Form.Item
                name="client_secret"
                label="客户端密码 (Client Secret)"
                rules={[{ required: true, message: '请输入 Client Secret' }]}
              >
                <Input.Password placeholder="输入客户端密码值" prefix={<KeyOutlined />} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={24}>
              <Form.Item
                name="tenant_id"
                label="租户 ID"
                initialValue="common"
                help="个人账号通常使用 'common'，组织账号请填写具体的 Tenant ID"
              >
                <Input placeholder="common" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item>
            <Button type="primary" htmlType="submit" block size="large">
              提交并前往授权
            </Button>
          </Form.Item>
        </Form>
      </Drawer>
    </div>
  );
};

export default Accounts;
