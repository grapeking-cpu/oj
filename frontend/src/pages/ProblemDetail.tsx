import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { Card, Tag, Button, Descriptions, Spin, Divider, Space } from 'antd'
import ReactMarkdown from 'react-markdown'
import remarkMath from 'remark-math'
import rehypeKatex from 'rehype-katex'
import { getProblem, type ProblemDetail as ProblemDetailType } from '@/api'

export default function ProblemDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [problem, setProblem] = useState<ProblemDetailType | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadProblem()
  }, [id])

  const loadProblem = async () => {
    if (!id) return
    setLoading(true)
    try {
      const data = await getProblem(parseInt(id))
      setProblem(data)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 50 }}><Spin size="large" /></div>
  }

  if (!problem) {
    return <div>题目不存在</div>
  }

  const getDifficultyName = (level: number) => {
    const names = ['入门', '简单', '中等', '困难', '极难']
    return names[level - 1] || '未知'
  }

  return (
    <div style={{ padding: 24, maxWidth: 1200, margin: '0 auto' }}>
      <Card
        title={
          <Space>
            <span>{problem.id}. {problem.title}</span>
            <Tag color="blue">{getDifficultyName(problem.difficulty)}</Tag>
          </Space>
        }
        extra={
          <Link to={`/problems/${problem.id}/submit`}>
            <Button type="primary">提交代码</Button>
          </Link>
        }
      >
        <Descriptions bordered column={2} size="small">
          <Descriptions.Item label="时间限制">{problem.time_limit} ms</Descriptions.Item>
          <Descriptions.Item label="内存限制">{problem.memory_limit} MB</Descriptions.Item>
          <Descriptions.Item label="难度">{getDifficultyName(problem.difficulty)}</Descriptions.Item>
          <Descriptions.Item label="通过率">{((problem.accept_rate || 0) * 100).toFixed(1)}%</Descriptions.Item>
        </Descriptions>

        <Divider />

        <div className="markdown-body">
          <h2>题目描述</h2>
          <ReactMarkdown remarkPlugins={[remarkMath]} rehypePlugins={[rehypeKatex]}>
            {problem.description || ''}
          </ReactMarkdown>

          <h2>输入格式</h2>
          <ReactMarkdown remarkPlugins={[remarkMath]} rehypePlugins={[rehypeKatex]}>
            {problem.input_format || ''}
          </ReactMarkdown>

          <h2>输出格式</h2>
          <ReactMarkdown remarkPlugins={[remarkMath]} rehypePlugins={[rehypeKatex]}>
            {problem.output_format || ''}
          </ReactMarkdown>

          {problem.sample_io && problem.sample_io.length > 0 && (
            <>
              <h2>样例输入/输出</h2>
              {problem.sample_io.map((sample, idx) => (
                <div key={idx} style={{ marginBottom: 16 }}>
                  <pre style={{ background: '#f5f5f5', padding: 10, borderRadius: 4 }}>
                    <strong>样例输入 {idx + 1}:</strong>
                    <br />
                    {sample.input}
                  </pre>
                  <pre style={{ background: '#f5f5f5', padding: 10, borderRadius: 4 }}>
                    <strong>样例输出 {idx + 1}:</strong>
                    <br />
                    {sample.output}
                  </pre>
                </div>
              ))}
            </>
          )}

          {problem.hint && (
            <>
              <h2>提示</h2>
              <ReactMarkdown remarkPlugins={[remarkMath]} rehypePlugins={[rehypeKatex]}>
                {problem.hint}
              </ReactMarkdown>
            </>
          )}
        </div>
      </Card>

      <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.css" />
    </div>
  )
}
