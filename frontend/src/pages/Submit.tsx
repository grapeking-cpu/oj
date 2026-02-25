import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Select, Button, message, Space } from 'antd'
import Editor from '@monaco-editor/react'
import { getLanguageList, createSubmit, type Language } from '@/api'

export default function Submit() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [languages, setLanguages] = useState<Language[]>([])
  const [code, setCode] = useState('')
  const [selectedLang, setSelectedLang] = useState<number>()

  useEffect(() => {
    loadLanguages()
  }, [])

  const loadLanguages = async () => {
    try {
      const data = await getLanguageList()
      setLanguages(data)
      if (data.length > 0) {
        setSelectedLang(data[0].id)
        setCode(`// 请输入 ${data[0].name} 代码`)
      }
    } catch (err) {
      console.error(err)
    }
  }

  const handleSubmit = async () => {
    if (!selectedLang || !code) {
      message.error('请选择语言并输入代码')
      return
    }

    setLoading(true)
    try {
      const result = await createSubmit({
        problem_id: parseInt(id!),
        language_id: selectedLang,
        code,
      })
      message.success('提交成功')
      navigate(`/submit/${result.submit_id}`)
    } catch (err: any) {
      message.error(err.message || '提交失败')
    } finally {
      setLoading(false)
    }
  }

  const handleLanguageChange = (langId: number) => {
    setSelectedLang(langId)
    const lang = languages.find(l => l.id === langId)
    if (lang) {
      setCode(`// 请输入 ${lang.name} 代码\n`)
    }
  }

  return (
    <div style={{ padding: 24 }}>
      <Card
        title="提交代码"
        extra={
          <Space>
            <Select
              value={selectedLang}
              onChange={handleLanguageChange}
              style={{ width: 150 }}
              options={languages.map(l => ({ label: l.name, value: l.id }))}
            />
            <Button type="primary" onClick={handleSubmit} loading={loading}>
              提交
            </Button>
          </Space>
        }
      >
        <Editor
          height="500px"
          language={languages.find(l => l.id === selectedLang)?.slug || 'plaintext'}
          theme="vs-dark"
          value={code}
          onChange={(value) => setCode(value || '')}
          options={{
            minimap: { enabled: false },
            fontSize: 14,
            lineNumbers: 'on',
            scrollBeyondLastLine: false,
          }}
        />
      </Card>
    </div>
  )
}
