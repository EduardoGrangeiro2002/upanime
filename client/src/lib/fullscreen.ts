import type { CSSProperties } from "react"

export function canUseRealFullscreen(): boolean {
  if (typeof document === "undefined") return false
  return document.fullscreenEnabled === true
}

export function needsCssRotation(): boolean {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") return false
  return window.matchMedia("(orientation: portrait)").matches
}

export function pseudoFullscreenStyle(rotate: boolean): CSSProperties {
  if (!rotate) {
    return {
      position: "fixed",
      inset: 0,
      width: "100dvw",
      height: "100dvh",
      zIndex: 9999,
      background: "#000",
    }
  }
  return {
    position: "fixed",
    top: 0,
    left: 0,
    width: "100dvh",
    height: "100dvw",
    transform: "translate(100dvw, 0) rotate(90deg)",
    transformOrigin: "top left",
    zIndex: 9999,
    background: "#000",
  }
}
