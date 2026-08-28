// 输入配置预览单元格：懒加载单帧画面（点击才抓，支持手动刷新与 10s 自动轮询）
import { useState, useEffect, useRef, useCallback } from 'react'
import { Button, Switch, Tooltip, Space } from 'antd'
import { PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import { getInputConfigPreview } from '../../../api/input-config'
import type { InputConfig } from '../../../types/input-config'

interface Props {
  config: InputConfig
}

type Status = 'idle' | 'loading' | 'ok' | 'error'

export function InputPreviewCell({ config }: Props) {
  const [status, setStatus] = useState<Status>('idle')
  const [url, setUrl] = useState<string | undefined>()
  const [errorMsg, setErrorMsg] = useState<string | undefined>()
  const [autoRefresh, setAutoRefresh] = useState(false)
  const inFlightRef = useRef(false)

  const revokeUrl = useCallback(() => {
    if (url) URL.revokeObjectURL(url)
  }, [url])

  const fetchFrame = useCallback(async () => {
    if (inFlightRef.current) return
    inFlightRef.current = true
    revokeUrl()
    setStatus('loading')
    try {
      const blob = await getInputConfigPreview(config.id)
      const objectUrl = URL.createObjectURL(blob)
      setUrl(objectUrl)
      setErrorMsg(undefined)
      setStatus('ok')
    } catch (err) {
      setUrl(undefined)
      setErrorMsg(err instanceof Error ? err.message : '无法获取画面')
      setStatus('error')
    } finally {
      inFlightRef.current = false
    }
  }, [config.id, revokeUrl])

  // 自动刷新
  useEffect(() => {
    if (!autoRefresh || status !== 'ok') return
    const interval = setInterval(fetchFrame, 10_000)
    return () => clearInterval(interval)
  }, [autoRefresh, status, fetchFrame])

  // 卸载清理
  useEffect(() => {
    return () => revokeUrl()
  }, [revokeUrl])

  if (status === 'idle') {
    return (
      <Button
        size="small"
        type="link"
        icon={<PlayCircleOutlined />}
        onClick={fetchFrame}
      >
        预览
      </Button>
    )
  }

  if (status === 'loading') {
    return <span style={{ color: '#999', fontSize: 12 }}>加载中...</span>
  }

  if (status === 'error') {
    return (
      <Space size={4} direction="vertical" style={{ maxWidth: 140 }}>
        <span style={{ color: '#ff4d4f', fontSize: 12 }}>无法获取画面</span>
        <Space size={4}>
          <Tooltip title={errorMsg}>
            <Button size="small" type="text" danger onClick={fetchFrame}>
              重试
            </Button>
          </Tooltip>
        </Space>
      </Space>
    )
  }

  // status === 'ok'
  return (
    <Space size={4} direction="vertical" style={{ width: '100%' }}>
      <img
        src={url}
        alt={`${config.name} 预览`}
        style={{ height: 54, borderRadius: 4, display: 'block' }}
      />
      <Space size={4}>
        <Tooltip title="刷新">
          <Button size="small" type="text" icon={<ReloadOutlined />} onClick={fetchFrame} />
        </Tooltip>
        <Switch
          size="small"
          checked={autoRefresh}
          onChange={(checked) => setAutoRefresh(checked)}
          checkedChildren="自动"
          unCheckedChildren="自动"
        />
      </Space>
    </Space>
  )
}
