import { Card, Button, Table } from 'antd'
import { Link } from 'react-router-dom'

export default function AdminDashboard() {
  // 简化版管理后台
  const columns = [
    { title: '功能', dataIndex: 'name' },
    { title: '描述', dataIndex: 'desc' },
    { title: '操作', render: () => <Button type="link">管理</Button> },
  ]

  const data = [
    { name: '题目管理', desc: '创建、编辑、删除题目' },
    { name: '用户管理', desc: '查看、禁用用户' },
    { name: '比赛管理', desc: '创建比赛、管理参赛者' },
    { name: '评测队列', desc: '查看待评测任务' },
    { name: '系统配置', desc: '配置系统参数' },
  ]

  return (
    <div style={{ padding: 24 }}>
      <Card title="管理后台" extra={<Link to="/"><Button>返回首页</Button></Link>}>
        <Table columns={columns} dataSource={data} rowKey="name" pagination={false} />
      </Card>
    </div>
  )
}
