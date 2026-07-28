// D-05.5 / D-08.1 — 自制极简单线条插画：404 岔路
// 由 Plan 15-05 的 NotFound.tsx 以 <NotFoundMascot className="..." /> 消费，API 保持稳定。
// 无外部插画库依赖；stroke 使用 currentColor，由调用方通过 CSS color 控制。

import type { CSSProperties } from 'react'

interface IllustrationProps {
  className?: string
  style?: CSSProperties
}

export default function NotFoundMascot({ className, style }: IllustrationProps) {
  return (
    <svg
      viewBox="0 0 200 140"
      className={className}
      style={style}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.25"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {/* 地平线 */}
      <line x1="26" y1="42" x2="174" y2="42" strokeDasharray="4 7" />

      {/* 主干道路边缘 */}
      <path d="M78 126 L94 78" />
      <path d="M118 126 L106 78" />

      {/* 左侧岔路（断头） */}
      <path d="M94 78 L64 50" />
      <path d="M106 78 L78 44" />

      {/* 右侧岔路 */}
      <path d="M94 78 L124 44" />
      <path d="M106 78 L138 50" />

      {/* 中央虚线路标 */}
      <line x1="100" y1="122" x2="100" y2="82" strokeDasharray="6 8" />

      {/* 左路尽头：断头墩 */}
      <line x1="66" y1="40" x2="80" y2="52" />
      <line x1="80" y1="40" x2="66" y2="52" />

      {/* 单点高亮：右路尽头的会议室小旗 */}
      <line x1="132" y1="46" x2="132" y2="24" stroke="#0F766E" />
      <path d="M132 24 h18 l-6 6 l6 6 h-18" stroke="#0F766E" />

      {/* 岔口标记 */}
      <circle cx="100" cy="78" r="3" stroke="#0F766E" />
    </svg>
  )
}
