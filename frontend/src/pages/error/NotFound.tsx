// 404 页面 (D-08.1) — antd Result + 自制 SVG 插画 + 中文文案 + stagger 渐入
// 不引入新的 ErrorBoundary (D-08.2)；错误 UX 仍由 antd message + useErrorHandler 驱动。

import { Result, Button } from 'antd'
import { useNavigate } from 'react-router-dom'
import { m } from 'framer-motion'
import NotFoundMascot from '../../assets/illustrations/NotFoundMascot'
import { staggerContainer, staggerItem } from '../../motion/motionConfig'
import { designTokens } from '../../styles/theme'
import styles from './NotFound.module.css'

export default function NotFound() {
  const navigate = useNavigate()
  return (
    <div className={styles.container}>
      <m.div
        variants={staggerContainer}
        initial="hidden"
        animate="visible"
        className={styles.content}
      >
        <m.div variants={staggerItem} className={styles.iconWrap}>
          <NotFoundMascot
            style={{
              width: 200,
              height: 140,
              color: designTokens.colors.muted,
            }}
          />
        </m.div>
        <m.div variants={staggerItem}>
          <Result
            status="404"
            title={<span style={{ color: designTokens.colors.text.primary }}>页面去开会了</span>}
            subTitle={<span style={{ color: designTokens.colors.text.secondary }}>这条路径没找到，回到首页继续</span>}
            extra={
              <Button type="primary" size="large" onClick={() => navigate('/dashboard')}>
                返回首页
              </Button>
            }
          />
        </m.div>
      </m.div>
    </div>
  )
}