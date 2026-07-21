import { useCallback, useEffect, useState } from "react"
import { canUseRealFullscreen, measureViewport, type Viewport } from "@/lib/fullscreen"

export function usePseudoFullscreen(resetKey: string) {
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [viewport, setViewport] = useState<Viewport | null>(null)

  useEffect(() => {
    setIsFullscreen(false)
  }, [resetKey])

  useEffect(() => {
    if (!isFullscreen) return
    const update = () => setViewport(measureViewport())
    update()
    window.visualViewport?.addEventListener("resize", update)
    window.addEventListener("resize", update)
    window.addEventListener("orientationchange", update)
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = "hidden"
    const restoreThemeColor = paintBrowserChromeBlack()
    return () => {
      window.visualViewport?.removeEventListener("resize", update)
      window.removeEventListener("resize", update)
      window.removeEventListener("orientationchange", update)
      document.body.style.overflow = previousOverflow
      restoreThemeColor()
    }
  }, [isFullscreen])

  const toggle = useCallback(() => setIsFullscreen((prev) => !prev), [])
  const exit = useCallback(() => setIsFullscreen(false), [])

  return { isFullscreen, viewport, toggle, exit, usesRealFullscreen: canUseRealFullscreen() }
}

function paintBrowserChromeBlack(): () => void {
  const existing = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')
  if (existing) {
    const previous = existing.getAttribute("content")
    existing.setAttribute("content", "#000000")
    return () => {
      if (previous === null) return
      existing.setAttribute("content", previous)
    }
  }
  const meta = document.createElement("meta")
  meta.name = "theme-color"
  meta.content = "#000000"
  document.head.appendChild(meta)
  return () => meta.remove()
}
