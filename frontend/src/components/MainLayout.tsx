import { Layout, Menu, Button, Space, Avatar, Dropdown } from 'antd'
import { Link, Outlet, useNavigate } from 'react-router-dom'
import { UserOutlined, BookOutlined, TrophyOutlined } from '@ant-design/icons'
import { useAuth } from '@/context/AuthContext'

const { Header, Content, Footer } = Layout

export default function MainLayout() {
  const { user, isAuthenticated, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const userMenu = {
    items: [
      { key: 'profile', label: '个人中心', onClick: () => navigate('/user') },
      { type: 'divider' as const },
      { key: 'logout', label: '退出登录', onClick: handleLogout },
    ],
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center', background: '#001529', padding: '0 24px' }}>
        <div style={{ color: '#fff', fontSize: 20, fontWeight: 'bold', marginRight: 40 }}>
          <Link to="/" style={{ color: '#fff' }}>OJ 评测平台</Link>
        </div>
        <Menu
          theme="dark"
          mode="horizontal"
          defaultSelectedKeys={['problems']}
          items={[
            { key: 'problems', label: <Link to="/problems">题目</Link>, icon: <BookOutlined /> },
            { key: 'contests', label: <Link to="/contests">比赛</Link>, icon: <TrophyOutlined /> },
          ]}
          style={{ flex: 1 }}
        />
        <Space>
          {isAuthenticated ? (
            <Dropdown menu={userMenu} placement="bottomRight">
              <Space style={{ cursor: 'pointer' }}>
                <Avatar icon={<UserOutlined />} src={user?.avatar} />
                <span style={{ color: '#fff' }}>{user?.username}</span>
              </Space>
            </Dropdown>
          ) : (
            <Link to="/login">
              <Button type="primary">登录</Button>
            </Link>
          )}
        </Space>
      </Header>
      <Content style={{ padding: '0', background: '#f0f2f5' }}>
        <div style={{ background: '#fff', minHeight: 'calc(100vh - 64px - 70px)' }}>
          <Outlet />
        </div>
      </Content>
      <Footer style={{ textAlign: 'center', background: '#fff' }}>
        OJ 评测平台 ©2024 - 构建你的编程能力
      </Footer>
    </Layout>
  )
}
