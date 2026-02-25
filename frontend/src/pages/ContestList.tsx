import { useState, useEffect } from 'react'
import { Table, Tag, Button } from 'antd'
import { Link } from 'react-router-dom'
import { getContestList, type Contest } from '@/api'

export default function ContestList() {
  const [contests, setContests] = useState<Contest[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    loadContests()
  }, [])

  const loadContests = async () => {
    setLoading(true)
    try {
      const data = await getContestList()
      setContests(data)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = { upcoming: 'blue', running: 'green', ended: 'default' }
    return colors[status] || 'default'
  }

  const getStatusName = (status: string) => {
    const names: Record<string, string> = { upcoming: '即将开始', running: '进行中', ended: '已结束' }
    return names[status] || status
  }

  const columns = [
    { title: '#', dataIndex: 'id', width: 60 },
    { title: '比赛名称', dataIndex: 'title', render: (title: string, r: Contest) => <Link to={`/contests/${r.id}`}>{title}</Link> },
    { title: '类型', dataIndex: 'type', width: 80 },
    { title: '状态', dataIndex: 'status', width: 100, render: (s: string) => <Tag color={getStatusColor(s)}>{getStatusName(s)}</Tag> },
    { title: '开始时间', dataIndex: 'start_time', width: 180 },
    { title: '结束时间', dataIndex: 'end_time', width: 180 },
    { title: '操作', width: 120, render: (_: any, r: Contest) => <Link to={`/contests/${r.id}`}><Button type="link">查看</Button></Link> },
  ]

  return (
    <div style={{ padding: 24 }}>
      <Table columns={columns} dataSource={contests} loading={loading} rowKey="id" />
    </div>
  )
}
