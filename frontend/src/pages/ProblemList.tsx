import { useState, useEffect } from 'react'
import { Table, Tag, Space, Button, Input, Select, Pagination, Card } from 'antd'
import { Link } from 'react-router-dom'
import { getProblemList, Problem } from '@/api'

const { Search } = Input

export default function ProblemList() {
  const [problems, setProblems] = useState<Problem[]>([])
  const [loading, setLoading] = useState(false)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [difficulty, setDifficulty] = useState<number | undefined>()
  const [search, setSearch] = useState('')

  useEffect(() => {
    loadProblems()
  }, [page, difficulty])

  const loadProblems = async () => {
    setLoading(true)
    try {
      const { list, total } = await getProblemList({ page, page_size: pageSize, difficulty, search })
      setProblems(list)
      setTotal(total)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = (value: string) => {
    setSearch(value)
    setPage(1)
    loadProblems()
  }

  const getDifficultyColor = (level: number) => {
    const colors = ['green', 'cyan', 'blue', 'orange', 'red']
    return colors[level - 1] || 'default'
  }

  const getDifficultyName = (level: number) => {
    const names = ['入门', '简单', '中等', '困难', '极难']
    return names[level - 1] || '未知'
  }

  const columns = [
    {
      title: '题号',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: '题目',
      dataIndex: 'title',
      render: (title: string, record: Problem) => (
        <Link to={`/problems/${record.id}`}>{title}</Link>
      ),
    },
    {
      title: '难度',
      dataIndex: 'difficulty',
      width: 100,
      render: (level: number) => (
        <Tag color={getDifficultyColor(level)}>{getDifficultyName(level)}</Tag>
      ),
    },
    {
      title: '标签',
      dataIndex: 'tags',
      render: (tags: string[]) => (
        <Space>
          {tags?.slice(0, 3).map(tag => (
            <Tag key={tag}>{tag}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '通过率',
      dataIndex: 'accept_rate',
      width: 100,
      render: (rate: number) => `${(rate * 100).toFixed(1)}%`,
    },
    {
      title: '操作',
      width: 120,
      render: (_: any, record: Problem) => (
        <Space>
          <Link to={`/problems/${record.id}`}>
            <Button type="link">做题</Button>
          </Link>
          <Link to={`/problems/${record.id}/submit`}>
            <Button type="link">提交</Button>
          </Link>
        </Space>
      ),
    },
  ]

  return (
    <div style={{ padding: 24 }}>
      <Card>
        <Space style={{ marginBottom: 16 }}>
          <Search placeholder="搜索题目" onSearch={handleSearch} style={{ width: 200 }} />
          <Select
            placeholder="难度"
            allowClear
            style={{ width: 120 }}
            onChange={setDifficulty}
            options={[
              { label: '入门', value: 1 },
              { label: '简单', value: 2 },
              { label: '中等', value: 3 },
              { label: '困难', value: 4 },
              { label: '极难', value: 5 },
            ]}
          />
          <Button onClick={loadProblems}>刷新</Button>
        </Space>

        <Table
          columns={columns}
          dataSource={problems}
          loading={loading}
          rowKey="id"
          pagination={false}
        />

        <div style={{ marginTop: 16, textAlign: 'right' }}>
          <Pagination
            current={page}
            pageSize={pageSize}
            total={total}
            onChange={setPage}
          />
        </div>
      </Card>
    </div>
  )
}
