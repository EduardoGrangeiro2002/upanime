import type { CSSProperties } from "react"

export interface Viewport {
  width: number
  height: number
}

export function canUseRealFullscreen(): boolean {
  if (typeof document === "undefined") return false
  return document.fullscreenEnabled === true
}

export function measureViewport(): Viewport {
  const visual = window.visualViewport
  if (visual) return { width: Math.round(visual.width), height: Math.round(visual.height) }
  return { width: window.innerWidth, height: window.innerHeight }
}

export function pseudoBackdropStyle(): CSSProperties {
  return { position: "fixed", inset: 0, zIndex: 9998, background: "#000" }
}

export function pseudoFullscreenStyle(viewport: Viewport): CSSProperties {
  if (viewport.width >= viewport.height) {
    return {
      position: "fixed",
      top: 0,
      left: 0,
      width: viewport.width,
      height: viewport.height,
      zIndex: 9999,
      background: "#000",
    }
  }
  return {
    position: "fixed",
    top: 0,
    left: 0,
    width: viewport.height,
    height: viewport.width,
    transform: `translate(${viewport.width}px, 0px) rotate(90deg)`,
    transformOrigin: "top left",
    zIndex: 9999,
    background: "#000",
  }
}
