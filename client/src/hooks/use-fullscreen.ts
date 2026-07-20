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
    return () => {
      window.visualViewport?.removeEventListener("resize", update)
      window.removeEventListener("resize", update)
      window.removeEventListener("orientationchange", update)
      document.body.style.overflow = previousOverflow
    }
  }, [isFullscreen])

  const toggle = useCallback(() => setIsFullscreen((prev) => !prev), [])
  const exit = useCallback(() => setIsFullscreen(false), [])

  return { isFullscreen, viewport, toggle, exit, usesRealFullscreen: canUseRealFullscreen() }
}
