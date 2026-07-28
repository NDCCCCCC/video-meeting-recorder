// Root motion provider — wraps the app ONCE.
// LazyMotion + domAnimation: tree-shakeable motion runtime (~12KB gz vs ~25KB for full motion).
// strict: forces children to use m.* (lowercase) instead of motion.* (which would
// bypass LazyMotion and pull the full ~25KB bundle). m + AnimatePresence are imported
// from the main 'framer-motion' package; framer-motion 12.34.0's '/m' subpath exports
// only element-named components, not the `m` namespace. With sideEffects:false +
// LazyMotion, the main-package `m` proxy stays tree-shaken.
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
