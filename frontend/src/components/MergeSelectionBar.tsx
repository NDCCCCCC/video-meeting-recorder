// 合并选择栏 - 拖拽排序底部栏
// 支持拖拽重新排序幻灯片顺序

import { Button, Space, Spin, Image } from 'antd'
import {
  CheckOutlined,
  CloseOutlined,
} from '@ant-design/icons'
import type { SelectedSlide } from '../types/ppt'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  horizontalListSortingStrategy,
  useSortable,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useCallback } from 'react'

interface MergeSelectionBarProps {
  selectedSlides: SelectedSlide[]
  onReorder: (slides: SelectedSlide[]) => void
  onRemove: (id: string) => void
  onConfirm: () => void
  onCancel: () => void
  isMerging: boolean
}

// 可排序的幻灯片项
interface SortableSlideProps {
  slide: SelectedSlide
  onRemove: (id: string) => void
}

function SortableSlide({ slide, onRemove }: SortableSlideProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: slide.id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      className="sortable-slide"
    >
      <div
        style={{
          position: 'relative',
          width: 120,
          height: 68,
          borderRadius: 4,
          overflow: 'hidden',
          border: '1px solid #d9d9d9',
          cursor: 'grab',
          background: '#ffffff',
        }}
      >
        <Image
          src={slide.thumbnail_url}
          alt={`幻灯片 ${slide.slide_number}`}
          width={120}
          height={68}
          preview={false}
          style={{ objectFit: 'cover' }}
        />

        {/* 来源标签 */}
        <div
          style={{
            position: 'absolute',
            bottom: 0,
            left: 0,
            right: 0,
            background: 'rgba(0, 0, 0, 0.6)',
            color: 'white',
            fontSize: 10,
            padding: '2px 4px',
            textAlign: 'center',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {slide.source_name}
        </div>

        {/* 移除按钮 */}
        <Button
          type="text"
          size="small"
          icon={<CloseOutlined />}
          onClick={(e) => {
            e.stopPropagation()
            onRemove(slide.id)
          }}
          aria-label={`移除幻灯片 ${slide.slide_number}`}
          style={{
            position: 'absolute',
            top: 2,
            right: 2,
            width: 20,
            height: 20,
            minWidth: 20,
            padding: 0,
            background: 'rgba(0, 0, 0, 0.6)',
            color: 'white',
            borderRadius: '50%',
          }}
        />
      </div>
    </div>
  )
}

export default function MergeSelectionBar({
  selectedSlides,
  onReorder,
  onRemove,
  onConfirm,
  onCancel,
  isMerging,
}: MergeSelectionBarProps) {
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  )

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event

      if (over && active.id !== over.id) {
        const oldIndex = selectedSlides.findIndex((s) => s.id === active.id)
        const newIndex = selectedSlides.findIndex((s) => s.id === over.id)

        onReorder(arrayMove(selectedSlides, oldIndex, newIndex))
      }
    },
    [selectedSlides, onReorder]
  )

  return (
    <div
      style={{
        borderTop: '2px solid #1890ff',
        background: '#ffffff',
        padding: 16,
      }}
    >
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 8,
        }}
      >
        <span
          style={{
            fontSize: 16,
            fontWeight: 500,
          }}
        >
          已选择 {selectedSlides.length}/200 页
          {selectedSlides.length >= 200 && (
            <span style={{ color: '#ff4d4f', marginLeft: 8 }}>
              (已达上限)
            </span>
          )}
        </span>
        <Space>
          <Button onClick={onCancel} disabled={isMerging}>
            取消合并
          </Button>
          <Button
            type="primary"
            onClick={onConfirm}
            disabled={selectedSlides.length === 0 || selectedSlides.length > 200 || isMerging}
            icon={isMerging ? <Spin size="small" /> : <CheckOutlined />}
          >
            {isMerging ? '合并中...' : '确认合并'}
          </Button>
        </Space>
      </div>

      {selectedSlides.length > 0 && (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext
            items={selectedSlides.map((s) => s.id)}
            strategy={horizontalListSortingStrategy}
          >
            <div
              style={{
                display: 'flex',
                gap: 8,
                overflowX: 'auto',
                paddingBottom: 8,
              }}
            >
              {selectedSlides.map((slide) => (
                <SortableSlide
                  key={slide.id}
                  slide={slide}
                  onRemove={onRemove}
                />
              ))}
            </div>
          </SortableContext>
        </DndContext>
      )}
    </div>
  )
}
