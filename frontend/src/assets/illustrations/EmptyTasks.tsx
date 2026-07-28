// D-05.5 — 自制极简单线条插画：录制任务空状态
// 无外部插画库依赖；stroke 使用 currentColor，由调用方通过 CSS color 控制。

import type { CSSProperties } from 'react'

interface IllustrationProps {
  className?: string
  style?: CSSProperties
}

export default function EmptyTasks({ className, style }: IllustrationProps) {
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
      {/* 摄像机机身 */}
      <rect x="44" y="44" width="78" height="50" rx="6" />
      {/* 镜头锥 */}
      <path d="M122 62 l26 -13 v42 l-26 -13 Z" />
      {/* 双磁带盘 */}
      <circle cx="72" cy="70" r="10" />
      <circle cx="72" cy="70" r="2" />
      <circle cx="100" cy="70" r="10" />
      <circle cx="100" cy="70" r="2" />

      {/* 单点高亮：录制指示灯 */}
      <circle cx="56" cy="54" r="3.5" stroke="#0F766E" />
      <path d="M62 50 a7 7 0 0 1 0 8" stroke="#0F766E" />

      {/* 三脚架 */}
      <line x1="83" y1="94" x2="83" y2="108" />
      <line x1="83" y1="108" x2="66" y2="118" />
      <line x1="83" y1="108" x2="100" y2="118" />

      {/* 地面虚线 */}
      <line x1="48" y1="122" x2="152" y2="122" strokeDasharray="4 7" />
    </svg>
  )
}
