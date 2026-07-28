// D-05.5 — 自制极简单线条插画：审计日志空状态
// 无外部插画库依赖；stroke 使用 currentColor，由调用方通过 CSS color 控制。

import type { CSSProperties } from 'react'

interface IllustrationProps {
  className?: string
  style?: CSSProperties
}

export default function EmptyAudit({ className, style }: IllustrationProps) {
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
      {/* 文档轮廓 + 折角 */}
      <path d="M60 22 h54 l22 22 v72 a4 4 0 0 1 -4 4 H64 a4 4 0 0 1 -4 -4 Z" />
      <path d="M114 22 v18 a4 4 0 0 0 4 4 h18" />

      {/* 时间轴主干 */}
      <line x1="78" y1="60" x2="78" y2="102" />

      {/* 时间轴节点 + 记录行 */}
      <circle cx="78" cy="60" r="3.5" />
      <line x1="90" y1="60" x2="120" y2="60" />
      <circle cx="78" cy="81" r="3.5" stroke="#0F766E" />
      <line x1="90" y1="81" x2="124" y2="81" stroke="#0F766E" />
      <circle cx="78" cy="102" r="3.5" />
      <line x1="90" y1="102" x2="112" y2="102" />

      {/* 地面虚线 */}
      <line x1="52" y1="128" x2="148" y2="128" strokeDasharray="4 7" />
    </svg>
  )
}
