// D-05.5 — 自制极简单线条插画：文件列表空状态
// 无外部插画库依赖；stroke 使用 currentColor，由调用方通过 CSS color 控制。

import type { CSSProperties } from 'react'

interface IllustrationProps {
  className?: string
  style?: CSSProperties
}

export default function EmptyFiles({ className, style }: IllustrationProps) {
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
      {/* 后方倾斜的纸张 */}
      <rect x="74" y="22" width="52" height="42" rx="3" transform="rotate(-7 100 43)" />
      {/* 前方纸张 + 内容留白线 */}
      <rect x="74" y="22" width="52" height="42" rx="3" />
      <line x1="84" y1="34" x2="112" y2="34" />
      <line x1="84" y1="43" x2="106" y2="43" />
      <line x1="84" y1="52" x2="116" y2="52" stroke="#0F766E" />

      {/* 文件夹背板 */}
      <path d="M46 96 V50 a4 4 0 0 1 4 -4 h20 l6 8 h48 a4 4 0 0 1 4 4 v6" />
      {/* 文件夹前板（敞开） */}
      <path d="M38 62 h124 a4 4 0 0 1 3.9 4.9 l-7.6 34 a5 5 0 0 1 -4.9 4.1 H47.6 a5 5 0 0 1 -4.9 -4.1 l-7.6 -34 A4 4 0 0 1 38 62 Z" />

      {/* 地面虚线 */}
      <line x1="50" y1="112" x2="150" y2="112" strokeDasharray="4 7" />

      {/* 单点高亮：右上角的"新增"提示 */}
      <circle cx="150" cy="40" r="7" stroke="#0F766E" />
      <line x1="150" y1="36" x2="150" y2="44" stroke="#0F766E" />
      <line x1="146" y1="40" x2="154" y2="40" stroke="#0F766E" />
    </svg>
  )
}
