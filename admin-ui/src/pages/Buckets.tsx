import React, { useEffect, useState } from 'react';
import { Table, Button, Modal, Form, Input, message, Popconfirm, Card, Typography } from 'antd';
import { PlusOutlined, DeleteOutlined, DatabaseOutlined } from '@ant-design/icons';
import { client } from '../api/client';

const { Title } = Typography;

interface Bucket {
  name: string;
  object_count: number;
  total_size: number;
  created_at: string;
}

const Buckets: React.FC = () => {
  const [buckets, setBuckets] = useState<Bucket[]>([]);
  const [loading, setLoading] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [form] = Form.useForm();

  const fetchBuckets = async () => {
    setLoading(true);
    try {
      const res = await client.get('/buckets');
      setBuckets(res.data.buckets || []);
    } catch (error) {
      message.error('获取存储桶列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBuckets();
  }, []);

  const handleCreate = async (values: { name: string }) => {
    try {
      await client.put(`/buckets/${values.name}`);
      message.success('存储桶创建成功');
      setIsModalOpen(false);
      form.resetFields();
      fetchBuckets();
    } catch (error) {
      message.error('创建存储桶失败');
    }
  };

  const handleDelete = async (name: string) => {
    try {
      await client.delete(`/buckets/${name}`);
      message.success('存储桶删除成功');
      fetchBuckets();
    } catch (error) {
      message.error('删除存储桶失败');
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
      title: '对象数量',
      dataIndex: 'object_count',
      key: 'object_count',
    },
    {
      title: '总大小 (Bytes)',
      dataIndex: 'total_size',
      key: 'total_size',
      render: (size: number) => (size / 1024 / 1024).toFixed(2) + ' MB',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date: string) => new Date(date).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Bucket) => (
        <Popconfirm
          title="删除存储桶"
          description="确定要删除这个存储桶吗？此操作不可恢复。"
          onConfirm={() => handleDelete(record.name)}
          okText="确定"
          cancelText="取消"
        >
          <Button type="link" danger icon={<DeleteOutlined />}>
            删除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={3} style={{ margin: 0 }}>存储桶管理</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setIsModalOpen(true)}>
          新建存储桶
        </Button>
      </div>

      <Card bordered={false} style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
        <Table 
          columns={columns} 
          dataSource={buckets} 
          rowKey="name" 
          loading={loading} 
          pagination={{ pageSize: 10 }}
          locale={{ emptyText: '暂无存储桶' }}
        />
      </Card>

      <Modal
        title="新建存储桶"
        open={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        footer={null}
      >
        <Form form={form} onFinish={handleCreate} layout="vertical">
          <Form.Item
            name="name"
            label="存储桶名称"
            rules={[{ required: true, message: '请输入存储桶名称' }]}
          >
            <Input prefix={<DatabaseOutlined />} placeholder="例如: my-backups" />
          </Form.Item>
          <Form.Item>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
              <Button onClick={() => setIsModalOpen(false)}>取消</Button>
              <Button type="primary" htmlType="submit">
                创建
              </Button>
            </div>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Buckets;
