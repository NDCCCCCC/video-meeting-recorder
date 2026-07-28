// Root motion provider — wraps the app ONCE.
// LazyMotion + domAnimation: tree-shakeable motion runtime (~12KB gz vs ~25KB for full motion).
// strict: forces children to use m.* (lowercase) from 'framer-motion/m' instead of motion.*
//   this is what keeps the bundle small per D-04.4.
// MotionConfig reducedMotion="user": honors OS-level prefers-reduced-motion automatically.
//
// IMPORTANT: import from 'framer-motion' (main package), NOT '/m' — the root needs
// LazyMotion/domAnimation/MotionConfig which are provider primitives, not animation components.

import { memo } from 'react'
import type { ReactNode } from 'react'
import { LazyMotion, MotionConfig, domAnimation } from 'framer-motion'

interface MotionProviderProps {
  children: ReactNode
}

function MotionProviderBase({ children }: MotionProviderProps) {
  return (
    <MotionConfig reducedMotion="user">
      <LazyMotion features={domAnimation} strict>
        {children}
      </LazyMotion>
    </MotionConfig>
  )
}

export const MotionProvider = memo(MotionProviderBase)
export default MotionProvider
