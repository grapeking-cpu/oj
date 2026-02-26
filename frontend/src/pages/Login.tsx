import { useState, useEffect } from 'react'
import { Form, Input, Button, Card, message, Tabs, Spin } from 'antd'
import { UserOutlined, LockOutlined, MailOutlined, ReloadOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/context/AuthContext'
import { getCaptcha } from '@/api'

export default function Login() {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [captchaLoading, setCaptchaLoading] = useState(false)
  const [captcha, setCaptcha] = useState<{ captcha_key: string; captcha_image: string } | null>(null)
  const { login, register } = useAuth()

  // 加载验证码
  const loadCaptcha = async () => {
    setCaptchaLoading(true)
    try {
      const res = await getCaptcha()
      // axios 拦截器已返回 response.data，所以直接是 { captcha_key, captcha_image }
      setCaptcha(res)
    } catch (err) {
      message.error('获取验证码失败')
    } finally {
      setCaptchaLoading(false)
    }
  }

  useEffect(() => {
    loadCaptcha()
  }, [])

  const onLogin = async (values: { username: string; password: string }) => {
    setLoading(true)
    try {
      await login(values.username, values.password)
      message.success('登录成功')
      // 使用 React Router 跳转，确保响应处理完成
      navigate('/')
    } catch (err: unknown) {
      const error = err as { message?: string }
      message.error(error?.message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  const onRegister = async (values: { username: string; email: string; password: string; captcha_code: string }) => {
    if (!captcha) {
      message.error('请先获取验证码')
      return
    }
    setLoading(true)
    try {
      await register({
        username: values.username,
        email: values.email,
        password: values.password,
        captcha_key: captcha.captcha_key,
        captcha_code: values.captcha_code,
      })
      message.success('注册成功')
      // 注册成功后跳转首页
      navigate('/')
    } catch (err: unknown) {
      const error = err as { message?: string }
      message.error(error?.message || '注册失败')
      // 注册失败后刷新验证码
      loadCaptcha()
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#f0f2f5' }}>
      <Card style={{ width: 420 }}>
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
              <Form onFinish={onRegister} layout="vertical">
                <Form.Item name="username" rules={[
                  { required: true, message: '请输入用户名' },
                  { min: 3, max: 50, message: '用户名 3-50 位' },
                  { pattern: /^[a-zA-Z0-9_]+$/, message: '仅限字母数字下划线' }
                ]}>
                  <Input prefix={<UserOutlined />} placeholder="用户名" />
                </Form.Item>
                <Form.Item name="email" rules={[
                  { required: true, type: 'email', message: '请输入有效邮箱' }
                ]}>
                  <Input prefix={<MailOutlined />} placeholder="邮箱" />
                </Form.Item>
                <Form.Item name="password" rules={[
                  { required: true, message: '请输入密码' },
                  { min: 8, max: 50, message: '密码 8-50 位，需含字母和数字' },
                  { pattern: /^(?=.*[a-zA-Z])(?=.*\d).+$/, message: '需含字母和数字' }
                ]}>
                  <Input.Password prefix={<LockOutlined />} placeholder="密码" />
                </Form.Item>
                <Form.Item name="captcha_code" rules={[{ required: true, message: '请输入验证码' }]}>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <Input placeholder="验证码" style={{ flex: 1 }} />
                    <Spin spinning={captchaLoading}>
                      <div
                        onClick={loadCaptcha}
                        style={{ cursor: 'pointer', border: '1px solid #d9d9d9', borderRadius: 6, overflow: 'hidden', width: 120, height: 40, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
                      >
                        {captcha?.captcha_image ? (
                          <img src={captcha.captcha_image} alt="验证码" style={{ maxWidth: '100%', maxHeight: '100%' }} />
                        ) : (
                          <ReloadOutlined />
                        )}
                      </div>
                    </Spin>
                  </div>
                </Form.Item>
                <Form.Item>
                  <Button type="primary" htmlType="submit" loading={loading} block>
                    注册
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
