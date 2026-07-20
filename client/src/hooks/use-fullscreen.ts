import { useCallback, useEffect, useState } from "react"
import { canUseRealFullscreen } from "@/lib/fullscreen"

export function usePseudoFullscreen(resetKey: string) {
  const [isFullscreen, setIsFullscreen] = useState(false)

  useEffect(() => {
    setIsFullscreen(false)
  }, [resetKey])

  const toggle = useCallback(() => setIsFullscreen((prev) => !prev), [])
  const exit = useCallback(() => setIsFullscreen(false), [])

  return { isFullscreen, toggle, exit, usesRealFullscreen: canUseRealFullscreen() }
}
