import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Card, Descriptions, Table, Tag, Button } from 'antd'
import { getContest, getContestRank } from '@/api'

export default function ContestDetail() {
  const { id } = useParams<{ id: string }>()
  const [contest, setContest] = useState<any>(null)
  const [rank, setRank] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadContest()
  }, [id])

  const loadContest = async () => {
    if (!id) return
    setLoading(true)
    try {
      const data = await getContest(parseInt(id))
      setContest(data)
      loadRank()
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const loadRank = async () => {
    if (!id) return
    try {
      const data = await getContestRank(parseInt(id))
      setRank(data)
    } catch (err) {
      console.error(err)
    }
  }

  if (loading || !contest) {
    return <div style={{ padding: 24 }}>加载中...</div>
  }

  return (
    <div style={{ padding: 24 }}>
      <Card title={contest.title} style={{ marginBottom: 16 }}>
        <Descriptions bordered column={2}>
          <Descriptions.Item label="类型">{contest.type}</Descriptions.Item>
          <Descriptions.Item label="状态"><Tag color={contest.status === 'running' ? 'green' : 'blue'}>{contest.status}</Tag></Descriptions.Item>
          <Descriptions.Item label="开始时间">{contest.start_time}</Descriptions.Item>
          <Descriptions.Item label="结束时间">{contest.end_time}</Descriptions.Item>
          <Descriptions.Item label="赛制">{contest.rule_type}</Descriptions.Item>
          <Descriptions.Item label="罚时">{contest.penalty_minutes} 分钟</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="题目列表" style={{ marginBottom: 16 }}>
        <Table
          dataSource={contest.problems || []}
          rowKey="letter"
          pagination={false}
          columns={[
            { title: '题号', dataIndex: 'letter' },
            { title: '题目', dataIndex: 'title' },
          ]}
        />
      </Card>

      <Card title="榜单">
        <Button onClick={loadRank} style={{ marginBottom: 16 }}>刷新榜单</Button>
        <Table
          dataSource={rank}
          rowKey="user_id"
          pagination={false}
          columns={[
            { title: '排名', dataIndex: 'rank', width: 80 },
            { title: '用户', dataIndex: 'user_id', width: 120 },
            { title: '解题数', dataIndex: 'score' },
            { title: '罚时', dataIndex: 'penalty' },
          ]}
        />
      </Card>
    </div>
  )
}
