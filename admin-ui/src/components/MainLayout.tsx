import React from 'react';
import { Layout, Menu, theme } from 'antd';
import { useNavigate, useLocation } from 'react-router-dom';
import { 
  LayoutDashboard, 
  Database, 
  Users, 
  ShieldCheck
} from 'lucide-react';

const { Header, Content, Sider } = Layout;

interface MainLayoutProps {
  children?: React.ReactNode;
}

const MainLayout: React.FC<MainLayoutProps> = ({ children }) => {
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken();
  
  const navigate = useNavigate();
  const location = useLocation();

  const items = [
    {
      key: '/dashboard',
      icon: <LayoutDashboard size={18} />,
      label: '系统概览',
    },
    {
      key: '/buckets',
      icon: <Database size={18} />,
      label: '存储桶管理',
    },
    {
      key: '/accounts',
      icon: <Users size={18} />,
      label: '账号管理',
    },
    {
      key: '/audit',
      icon: <ShieldCheck size={18} />,
      label: '审计与健康',
    },
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider breakpoint="lg" collapsedWidth="0" width={220}>
        <div style={{ height: 32, margin: 16, background: 'rgba(255, 255, 255, 0.2)', borderRadius: 6, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'white', fontWeight: 'bold', letterSpacing: '1px' }}>
          OneDrive Storage
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={items}
          onClick={({ key }) => navigate(key)}
          style={{ fontSize: '15px' }}
        />
      </Sider>
      <Layout>
        <Header style={{ padding: '0 24px', background: colorBgContainer, display: 'flex', alignItems: 'center', justifyContent: 'space-between', boxShadow: '0 1px 4px rgba(0,21,41,0.08)' }}>
           <h2 style={{ margin: 0, fontSize: '18px', fontWeight: 600 }}>OneDrive 分布式存储中间件</h2>
        </Header>
        <Content style={{ margin: '24px 16px 0' }}>
          <div
            style={{
              padding: 24,
              minHeight: 360,
              background: colorBgContainer,
              borderRadius: borderRadiusLG,
            }}
          >
            {children}
          </div>
        </Content>
        <Layout.Footer style={{ textAlign: 'center', color: '#888' }}>
          OneDrive Storage Middleware ©{new Date().getFullYear()} Created by xuecangming
        </Layout.Footer>
      </Layout>
    </Layout>
  );
};

export default MainLayout;
