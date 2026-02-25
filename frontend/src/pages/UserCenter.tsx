import { useState, useEffect } from 'react'
import { Card, Avatar, Descriptions, Button, List, Tag, Space } from 'antd'
import { UserOutlined, LogoutOutlined } from '@ant-design/icons'
import { useAuth } from '@/context/AuthContext'
import { getMySubmits } from '@/api'

export default function UserCenter() {
  const { user, logout } = useAuth()
  const [submits, setSubmits] = useState<any[]>([])

  useEffect(() => {
    loadSubmits()
  }, [])

  const loadSubmits = async () => {
    try {
      const { list } = await getMySubmits()
      setSubmits(list.slice(0, 10))
    } catch (err) {
      console.error(err)
    }
  }

  const getStatusColor = (status: string) => {
    const colors: any = { PENDING: 'orange', RUNNING: 'blue', FINISHED: 'green', CE: 'red', WA: 'red' }
    return colors[status] || 'default'
  }

  return (
    <div style={{ padding: 24, maxWidth: 1000, margin: '0 auto' }}>
      <Card>
        <Space>
          <Avatar size={64} icon={<UserOutlined />} src={user?.avatar} />
          <Descriptions title={user?.nickname || user?.username} column={2}>
            <Descriptions.Item label="用户名">{user?.username}</Descriptions.Item>
            <Descriptions.Item label="Rating">{user?.rating}</Descriptions.Item>
            <Descriptions.Item label="提交数">{user?.submit_count}</Descriptions.Item>
            <Descriptions.Item label="通过数">{user?.accept_count}</Descriptions.Item>
          </Descriptions>
          <Button icon={<LogoutOutlined />} onClick={logout}>退出登录</Button>
        </Space>
      </Card>

      <Card title="最近提交" style={{ marginTop: 16 }}>
        <List
          dataSource={submits}
          renderItem={item => (
            <List.Item>
              <Space>
                <Tag>#{item.submit_id?.slice(0, 8)}</Tag>
                <Tag>题目 {item.problem_id}</Tag>
                <Tag color={getStatusColor(item.judge_result?.status)}>{item.judge_result?.status || 'PENDING'}</Tag>
                <span>{item.judge_result?.score || 0} 分</span>
              </Space>
            </List.Item>
          )}
        />
      </Card>
    </div>
  )
}
