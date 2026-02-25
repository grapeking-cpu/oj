import { useState } from 'react'
import { Form, Input, Button, Card, message, Tabs } from 'antd'
import { UserOutlined, LockOutlined, MailOutlined } from '@ant-design/icons'
import { useAuth } from '@/context/AuthContext'

export default function Login() {
  const [loading, setLoading] = useState(false)
  const { login: authLogin } = useAuth()

  const onLogin = async (values: { username: string; password: string }) => {
    setLoading(true)
    try {
      await authLogin(values.username, values.password)
      message.success('登录成功')
    } catch (err: unknown) {
      const error = err as { message?: string }
      message.error(error?.message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#f0f2f5' }}>
      <Card style={{ width: 400 }}>
        <h1 style={{ textAlign: 'center', marginBottom: 24 }}>OJ 评测平台</h1>
        <Tabs items={[
          {
            key: 'login',
            label: '登录',
            children: (
              <Form onFinish={onLogin} layout="vertical">
                <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
                  <Input prefix={<UserOutlined />} placeholder="用户名" />
                </Form.Item>
                <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
                  <Input.Password prefix={<LockOutlined />} placeholder="密码" />
                </Form.Item>
                <Form.Item>
                  <Button type="primary" htmlType="submit" loading={loading} block>
                    登录
                  </Button>
                </Form.Item>
              </Form>
            )
          },
          {
            key: 'register',
            label: '注册',
            children: (
              <Form layout="vertical">
                <Form.Item rules={[{ required: true, message: '请输入用户名' }]}>
                  <Input prefix={<UserOutlined />} placeholder="用户名" />
                </Form.Item>
                <Form.Item rules={[{ required: true, type: 'email', message: '请输入邮箱' }]}>
                  <Input prefix={<MailOutlined />} placeholder="邮箱" />
                </Form.Item>
                <Form.Item rules={[{ required: true, min: 8, message: '密码至少8位' }]}>
                  <Input.Password prefix={<LockOutlined />} placeholder="密码" />
                </Form.Item>
                <Form.Item>
                  <Input placeholder="验证码" />
                </Form.Item>
                <Form.Item>
                  <Button type="primary" block disabled>
                    注册 (暂未开放)
                  </Button>
                </Form.Item>
              </Form>
            )
          }
        ]} />
      </Card>
    </div>
  )
}
