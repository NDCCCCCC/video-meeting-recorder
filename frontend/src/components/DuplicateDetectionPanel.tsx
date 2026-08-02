import React, { useState, useEffect } from 'react'
import {
  Modal,
  Card,
  Checkbox,
  Button,
  Space,
  Image,
  Alert,
  message,
  Progress,
  Tag,
  Typography,
} from 'antd'
import { DeleteOutlined, ScanOutlined, RollbackOutlined } from '@ant-design/icons'
import type { CheckboxChangeEvent } from 'antd/es/checkbox'
import { detectDuplicates, deleteSlides, rollbackPPT, type DuplicateGroup } from '../api/ppt'

const { Text } = Typography

// 从 API 错误提取用户可见消息（apiRequest 抛 Error；response.data 为防御性兼容访问）
function extractApiError(err: unknown, fallback: string): string {
  const e = err as { response?: { data?: { error?: string } }; message?: string }
  return e.response?.data?.error || e.message || fallback
}

interface DuplicateDetectionPanelProps {
  pptFileId: number
  visible: boolean
  onClose: () => void
  onSlidesDeleted?: () => void
}

const DuplicateDetectionPanel: React.FC<DuplicateDetectionPanelProps> = ({
  pptFileId,
  visible,
  onClose,
  onSlidesDeleted,
}) => {
  const [duplicateGroups, setDuplicateGroups] = useState<DuplicateGroup[]>([])
  const [selectedForDeletion, setSelectedForDeletion] = useState<Set<number>>(new Set())
  const [isScanning, setIsScanning] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [isRollingBack, setIsRollingBack] = useState(false)
  const [scanProgress, setScanProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)

  // Detect duplicates on mount
  useEffect(() => {
    if (visible) {
      handleDetectDuplicates()
    }
  }, [visible, pptFileId])

  const handleDetectDuplicates = async () => {
    setIsScanning(true)
    setError(null)
    setScanProgress(0)

    // Simulate progress
    const progressInterval = setInterval(() => {
      setScanProgress((prev) => Math.min(prev + 10, 90))
    }, 500)

    try {
      const response = await detectDuplicates(pptFileId)
      if (!response.data) {
        throw new Error('No data in response')
      }
      setDuplicateGroups(response.data.groups || [])
      setScanProgress(100)

      const groups = response.data.groups || []
      if (groups.length === 0) {
        message.info('未检测到重复幻灯片')
      } else {
        message.success(`检测到 ${groups.length} 组重复幻灯片`)
      }
    } catch (err) {
      const errorMsg = extractApiError(err, '检测失败')
      setError(errorMsg)
      message.error(errorMsg)
    } finally {
      clearInterval(progressInterval)
      setIsScanning(false)
    }
  }

  const handleSlideSelect = (slideNum: number, checked: boolean) => {
    setSelectedForDeletion((prev) => {
      const newSet = new Set(prev)
      if (checked) {
        newSet.add(slideNum)
      } else {
        newSet.delete(slideNum)
      }
      return newSet
    })
  }

  const handleSelectAllInGroup = (group: DuplicateGroup, checked: boolean) => {
    setSelectedForDeletion((prev) => {
      const newSet = new Set(prev)
      // Select all except the first one (keep original)
      const slidesToSelect = checked ? group.slides.slice(1) : group.slides
      slidesToSelect.forEach((slide) => {
        if (checked) {
          newSet.add(slide)
        } else {
          newSet.delete(slide)
        }
      })
      return newSet
    })
  }

  const handleDeleteSelected = async () => {
    if (selectedForDeletion.size === 0) {
      message.warning('请先选择要删除的幻灯片')
      return
    }

    Modal.confirm({
      title: '确认删除',
      content: `确定要删除选中的 ${selectedForDeletion.size} 张幻灯片吗？此操作将创建备份，可以通过回滚功能恢复。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        setIsDeleting(true)
        try {
          const slidesToDelete = Array.from(selectedForDeletion)
          await deleteSlides(pptFileId, slidesToDelete)
          message.success('幻灯片删除成功')
          setSelectedForDeletion(new Set())
          onSlidesDeleted?.()
          onClose()
        } catch (err) {
          const errorMsg = extractApiError(err, '删除失败')
          message.error(errorMsg)
        } finally {
          setIsDeleting(false)
        }
      },
    })
  }

  const handleRollback = async () => {
    Modal.confirm({
      title: '确认回滚',
      content: '确定要回滚到备份版本吗？此操作将撤销所有编辑。',
      okText: '回滚',
      okType: 'primary',
      cancelText: '取消',
      onOk: async () => {
        setIsRollingBack(true)
        try {
          await rollbackPPT(pptFileId)
          message.success('回滚成功')
          onSlidesDeleted?.()
          onClose()
        } catch (err) {
          const errorMsg = extractApiError(err, '回滚失败')
          message.error(errorMsg)
        } finally {
          setIsRollingBack(false)
        }
      },
    })
  }

  const getSimilarityColor = (similarity: number) => {
    if (similarity >= 0.98) return 'success'
    if (similarity >= 0.95) return 'warning'
    return 'default'
  }

  return (
    <Modal
      title={
        <Space>
          <ScanOutlined />
          重复幻灯片检测
        </Space>
      }
      open={visible}
      onCancel={onClose}
      width={1000}
      footer={
        <Space>
          <Button onClick={onClose}>关闭</Button>
          <Button icon={<ScanOutlined />} onClick={handleDetectDuplicates} loading={isScanning}>
            重新扫描
          </Button>
          <Button icon={<RollbackOutlined />} onClick={handleRollback} loading={isRollingBack}>
            回滚
          </Button>
          <Button
            type="primary"
            danger
            icon={<DeleteOutlined />}
            onClick={handleDeleteSelected}
            loading={isDeleting}
            disabled={selectedForDeletion.size === 0}
          >
            删除选中 ({selectedForDeletion.size})
          </Button>
        </Space>
      }
    >
      {error && (
        <Alert
          title="扫描失败"
          description={error}
          type="error"
          closable
          onClose={() => setError(null)}
          style={{ marginBottom: 16 }}
        />
      )}

      {isScanning && (
        <div style={{ marginBottom: 24 }}>
          <Text>正在扫描幻灯片...</Text>
          <Progress percent={scanProgress} status="active" />
        </div>
      )}

      {!isScanning && duplicateGroups.length === 0 && (
        <Alert
          title="未检测到重复幻灯片"
          description="所有幻灯片都是独特的，无需删除重复项。"
          type="success"
          showIcon
        />
      )}

      <Space orientation="vertical" style={{ width: '100%', maxHeight: '60vh', overflowY: 'auto' }}>
        {duplicateGroups.map((group, index) => (
          <Card
            key={index}
            type="inner"
            title={
              <Space>
                <Text strong>重复组 #{index + 1}</Text>
                <Tag color={getSimilarityColor(group.similarity)}>
                  相似度: {(group.similarity * 100).toFixed(1)}%
                </Tag>
                <Tag>SSIM: {group.ssim_score.toFixed(3)}</Tag>
                <Tag>pHash: {group.phash_distance}</Tag>
              </Space>
            }
            extra={
              <Checkbox
                checked={group.slides.slice(1).every((s) => selectedForDeletion.has(s))}
                onChange={(e) => handleSelectAllInGroup(group, e.target.checked)}
              >
                全选除首张外所有
              </Checkbox>
            }
          >
            <Space orientation="vertical" style={{ width: '100%' }}>
              {group.slides.map((slideNum) => {
                const isRecommended = slideNum !== group.slides[0]
                const filename = `slide_${String(slideNum).padStart(3, '0')}.jpg`
                return (
                  <div
                    key={slideNum}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 16,
                      padding: '8px',
                      borderRadius: '4px',
                      backgroundColor: isRecommended ? 'rgba(0, 0, 0, 0.02)' : 'transparent',
                    }}
                  >
                    <Checkbox
                      checked={selectedForDeletion.has(slideNum)}
                      onChange={(e: CheckboxChangeEvent) =>
                        handleSlideSelect(slideNum, e.target.checked)
                      }
                    />
                    <Image
                      width={120}
                      src={`/api/v1/ppts/${pptFileId}/slides/thumbnails/${filename}`}
                      preview={{
                        src: `/api/v1/ppts/${pptFileId}/slides/fullsize/${filename}`,
                      }}
                    />
                    <div style={{ flex: 1 }}>
                      <Text strong>幻灯片 #{slideNum}</Text>
                      {isRecommended && (
                        <Tag color="orange" style={{ marginLeft: 8 }}>
                          建议删除
                        </Tag>
                      )}
                    </div>
                  </div>
                )
              })}
            </Space>
          </Card>
        ))}
      </Space>
    </Modal>
  )
}

export default DuplicateDetectionPanel
