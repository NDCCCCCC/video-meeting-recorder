// 登录页动效背景组件 — Phase 16 D-02
// - 深青多层径向渐变（D-02.1）
// - 3 个模糊光晕缓慢漂浮（D-02.2）
// - SVG 点阵网格轻微抖动（D-02.3）
// - 鼠标光晕跟随（D-02.4）
// - 尊重 prefers-reduced-motion（D-02.7）
//
// 性能修复：鼠标跟随用 ref + requestAnimationFrame 直接写 DOM transform，
// 不走 React state，避免鼠标移动时整个登录卡/表单被反复重渲染导致点击/输入卡顿。
import { useEffect, useRef, type ReactNode } from 'react'
import styles from './LoginBackground.module.css'

export default function LoginBackground({ children }: { children?: ReactNode }) {
  const haloRef = useRef<HTMLDivElement>(null)
  const rafRef = useRef(0)

  useEffect(() => {
    const halo = haloRef.current
    if (!halo) return

    let lastEvent: MouseEvent | null = null
    const handleMouseMove = (e: MouseEvent) => {
      lastEvent = e
      // rAF 节流：一帧内只更新一次 DOM，避免高频 mousemove 触发重渲染/重排
      if (rafRef.current) return
      rafRef.current = requestAnimationFrame(() => {
        rafRef.current = 0
        const ev = lastEvent
        if (!ev) return
        const x = (ev.clientX / window.innerWidth) * 100
        const y = (ev.clientY / window.innerHeight) * 100
        halo.style.transform = `translate(${x - 50}vw, ${y - 50}vh)`
      })
    }

    window.addEventListener('mousemove', handleMouseMove, { passive: true })
    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current)
        rafRef.current = 0
      }
    }
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

      {/* 鼠标光晕跟随层：D-02.4（ref 直接写 transform，不触发 React 重渲染） */}
      <div className={styles.mouseHalo} ref={haloRef} aria-hidden="true" />

      {/* 内容层（登录卡） */}
      <div className={styles.content}>{children}</div>
    </div>
  )
}