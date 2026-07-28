// D-05.5 — 自制极简单线条插画：加载/网络错误状态
// 无外部插画库依赖；stroke 使用 currentColor，由调用方通过 CSS color 控制。

import type { CSSProperties } from 'react'

interface IllustrationProps {
  className?: string
  style?: CSSProperties
}

export default function ErrorNetwork({ className, style }: IllustrationProps) {
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
      {/* 左侧节点 */}
      <rect x="24" y="52" width="48" height="34" rx="5" />
      <line x1="34" y1="62" x2="52" y2="62" />
      <circle cx="36" cy="76" r="2.5" />
      <line x1="44" y1="76" x2="60" y2="76" />

      {/* 右侧节点 */}
      <rect x="128" y="52" width="48" height="34" rx="5" />
      <line x1="138" y1="62" x2="156" y2="62" />
      <circle cx="140" cy="76" r="2.5" />
      <line x1="148" y1="76" x2="164" y2="76" />

      {/* 断开的连线 */}
      <line x1="72" y1="69" x2="90" y2="69" />
      <line x1="110" y1="69" x2="128" y2="69" />

      {/* 单点高亮：断点 */}
      <line x1="94" y1="60" x2="88" y2="78" stroke="#0F766E" />
      <line x1="106" y1="60" x2="100" y2="78" stroke="#0F766E" />

      {/* 断点火花 */}
      <line x1="96" y1="50" x2="99" y2="44" />
      <line x1="103" y1="88" x2="106" y2="94" />

      {/* 地面虚线 */}
      <line x1="40" y1="112" x2="160" y2="112" strokeDasharray="4 7" />
    </svg>
  )
}
