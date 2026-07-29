// 登录页动效背景组件 — Phase 16 D-02
// - 深青多层径向渐变（D-02.1）
// - 3 个模糊光晕缓慢漂浮（D-02.2）
// - SVG 点阵网格轻微抖动（D-02.3）
// - 鼠标光晕跟随（D-02.4）
// - 尊重 prefers-reduced-motion（D-02.7）
import { useEffect, useState, type ReactNode } from 'react'
import styles from './LoginBackground.module.css'

export default function LoginBackground({ children }: { children?: ReactNode }) {
  const [mousePos, setMousePos] = useState({ x: 50, y: 50 })

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      const x = (e.clientX / window.innerWidth) * 100
      const y = (e.clientY / window.innerHeight) * 100
      setMousePos({ x, y })
    }
    window.addEventListener('mousemove', handleMouseMove)
    return () => window.removeEventListener('mousemove', handleMouseMove)
  }, [])

  return (
    <div className={styles.container}>
      {/* 背景层：D-02.1 深青多层径向渐变 */}
      <div className={styles.bgGradient} aria-hidden="true" />

      {/* 光晕层：D-02.2 3 个模糊光晕 */}
      <div className={styles.halo1} aria-hidden="true" />
      <div className={styles.halo2} aria-hidden="true" />
      <div className={styles.halo3} aria-hidden="true" />

      {/* 点阵网格层：D-02.3 SVG pattern 轻微抖动 */}
      <svg className={styles.dotGrid} aria-hidden="true" xmlns="http://www.w3.org/2000/svg">
        <defs>
          <pattern
            id="dot-pattern"
            x="0"
            y="0"
            width="40"
            height="40"
            patternUnits="userSpaceOnUse"
          >
            <circle cx="20" cy="20" r="1" fill="currentColor" />
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill="url(#dot-pattern)" />
      </svg>

      {/* 鼠标光晕跟随层：D-02.4 */}
      <div
        className={styles.mouseHalo}
        aria-hidden="true"
        style={{
          transform: `translate(${mousePos.x - 50}vw, ${mousePos.y - 50}vh)`,
        }}
      />

      {/* 内容层（登录卡） */}
      <div className={styles.content}>{children}</div>
    </div>
  )
}