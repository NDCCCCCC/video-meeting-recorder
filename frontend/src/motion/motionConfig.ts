// Shared framer-motion variants — single source of truth for timing.
// All variants read durations from designTokens.motion so D-03.5 timings stay consistent.
// framer-motion `transition.duration` expects SECONDS — divide ms tokens by 1000.
//
// NOTE on easing: framer-motion's `Easing` type accepts bezier 4-tuples (not CSS
// `cubic-bezier(...)` strings). theme.ts stores easing as CSS strings for CSS
// consumers; here we mirror those values as BezierDefinition tuples so the variants
// are fully typed. The two must stay in sync — see theme.ts motion.easing.
//   standard  = cubic-bezier(0.4, 0, 0.2, 1)   -> [0.4, 0, 0.2, 1]
//   decelerate= cubic-bezier(0, 0, 0.2, 1)     -> [0, 0, 0.2, 1]
//   accelerate= cubic-bezier(0.4, 0, 1, 1)     -> [0.4, 0, 1, 1]

import type { Variants } from 'framer-motion'
import { designTokens } from '../styles/theme'

const ms = (n: number) => n / 1000

// Bezier tuples mirroring designTokens.motion.easing.* (CSS strings live in theme.ts)
const easing = {
  standard: [0.4, 0, 0.2, 1] as const,
  decelerate: [0, 0, 0.2, 1] as const,
  accelerate: [0.4, 0, 1, 1] as const,
}

/**
 * Fade-in for blocks that should appear once on mount.
 * Used by: QuickActions, dashboard page wrapper, error/loading branches.
 */
export const fadeIn: Variants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      duration: ms(designTokens.motion.duration.base),
      ease: easing.standard,
    },
  },
}

/**
 * Stagger container — orchestrates delay between children.
 * Per CONTEXT.md specifics: 30ms start +20ms/step, cap 200ms.
 * staggerChildren=0.02 (20ms per step), delayChildren=0.03 (30ms start).
 * Per-item cap handled by framer-motion's automatic stagger ceiling.
 */
export const staggerContainer: Variants = {
  hidden: { opacity: 1 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.02,
      delayChildren: 0.03,
    },
  },
}

/**
 * Stagger item — child of staggerContainer.
 * Subtle 8px rise + fade, using decelerate easing (entering motion).
 */
export const staggerItem: Variants = {
  hidden: { opacity: 0, y: 8 },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      duration: ms(designTokens.motion.duration.base),
      ease: easing.decelerate,
    },
  },
}

/**
 * Route-level fade — used by AnimatePresence in App.tsx.
 * Fast duration (120ms) per D-04.3. Different easing for enter (standard)
 * vs exit (accelerate) so exits feel snappier.
 */
export const routeFade: Variants = {
  initial: {
    opacity: 0,
  },
  animate: {
    opacity: 1,
    transition: {
      duration: ms(designTokens.motion.duration.fast),
      ease: easing.standard,
    },
  },
  exit: {
    opacity: 0,
    transition: {
      duration: ms(designTokens.motion.duration.fast),
      ease: easing.accelerate,
    },
  },
}

export type MotionVariants = typeof fadeIn
