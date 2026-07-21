import { useEffect, useRef, useCallback, useState } from "react"
import { createPortal } from "react-dom"
import { MediaPlayer, MediaProvider, useMediaState, type MediaPlayerInstance } from "@vidstack/react"
import { PlayerControls } from "./player-controls"
import { usePseudoFullscreen } from "@/hooks/use-fullscreen"
import { pseudoBackdropStyle, pseudoFullscreenStyle } from "@/lib/fullscreen"
import { buildQualityOptions } from "@/lib/quality"
import type { Episode } from "@/api/types"

export function requestRealFullscreen(player: MediaPlayerInstance, active: boolean) {
  const request = active ? player.exitFullscreen() : player.enterFullscreen()
  request.catch(() => {})
}

interface VideoPlayerProps {
  src: string
  title: string
  episode: Episode
  resolveVariantUrl: (variant: string) => string
  autoPlay?: boolean
  onClose: () => void
  onPrevious?: () => void
  onNext?: () => void
  initialTime?: number
  onTimeUpdate?: (time: number, duration: number) => void
  onPause?: () => void
}

export function VideoPlayer({
  src,
  title,
  episode,
  resolveVariantUrl,
  autoPlay = false,
  onClose,
  onPrevious,
  onNext,
  initialTime,
  onTimeUpdate,
  onPause,
}: VideoPlayerProps) {
  const playerRef = useRef<MediaPlayerInstance>(null)
  const seekedRef = useRef(false)
  const pendingSeekRef = useRef<number | null>(null)
  const [source, setSource] = useState(src)
  const [activeQuality, setActiveQuality] = useState("original")
  const [menuOpen, setMenuOpen] = useState(false)
  const { isFullscreen, viewport, toggle, exit, usesRealFullscreen } = usePseudoFullscreen(episode.id)
  const mediaFullscreen = useMediaState("fullscreen", playerRef)

  const qualities = buildQualityOptions(episode)

  useEffect(() => {
    seekedRef.current = false
    pendingSeekRef.current = null
    setSource(src)
    setActiveQuality("original")
  }, [episode.id, src])

  const handleTimeUpdate = useCallback(() => {
    const player = playerRef.current
    if (!player) return
    onTimeUpdate?.(player.currentTime, player.duration)
  }, [onTimeUpdate])

  const handleCanPlay = useCallback(() => {
    const player = playerRef.current
    if (!player) return
    if (pendingSeekRef.current !== null) {
      player.currentTime = pendingSeekRef.current
      pendingSeekRef.current = null
      return
    }
    if (seekedRef.current || !initialTime || initialTime < 2) return
    player.currentTime = initialTime
    seekedRef.current = true
  }, [initialTime])

  const selectQuality = useCallback(
    (variant: string) => {
      const player = playerRef.current
      if (!player || variant === activeQuality) return
      pendingSeekRef.current = player.currentTime
      setActiveQuality(variant)
      setSource(resolveVariantUrl(variant))
    },
    [activeQuality, resolveVariantUrl],
  )

  const toggleFullscreen = useCallback(() => {
    const player = playerRef.current
    if (!player) return
    if (usesRealFullscreen) {
      requestRealFullscreen(player, mediaFullscreen)
      return
    }
    toggle()
  }, [toggle, usesRealFullscreen, mediaFullscreen])

  const controls = (
    <PlayerControls
      onClose={onClose}
      onPrevious={onPrevious}
      onNext={onNext}
      qualities={qualities}
      activeQuality={activeQuality}
      onSelectQuality={selectQuality}
      isFullscreen={usesRealFullscreen ? mediaFullscreen : isFullscreen}
      onToggleFullscreen={toggleFullscreen}
      menuOpen={menuOpen}
      onMenuOpenChange={setMenuOpen}
    />
  )

  const pseudoActive = isFullscreen && !usesRealFullscreen

  const slotRef = useRef<HTMLDivElement>(null)
  const slotRect = useSlotRect(slotRef, !pseudoActive)

  useEffect(() => {
    if (!pseudoActive) return
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || menuOpen) return
      exit()
    }
    document.addEventListener("keydown", onKey)
    return () => document.removeEventListener("keydown", onKey)
  }, [pseudoActive, menuOpen, exit])

  const playerStyle: React.CSSProperties =
    pseudoActive && viewport
      ? pseudoFullscreenStyle(viewport)
      : slotRect
        ? { position: "fixed", top: slotRect.top, left: slotRect.left, width: slotRect.width, height: slotRect.height, zIndex: 55 }
        : { position: "fixed", opacity: 0, pointerEvents: "none" }

  const playerLayer = (
    <div style={pseudoActive ? pseudoBackdropStyle() : undefined}>
      <div style={playerStyle} className="overflow-hidden rounded-lg bg-black">
        <MediaPlayer
          ref={playerRef}
          key={episode.id}
          src={{ src: source, type: "video/mp4" }}
          title={title}
          aspectRatio={pseudoActive ? undefined : "16/9"}
          playsInline
          autoPlay={autoPlay}
          fullscreenOrientation="landscape"
          onTimeUpdate={handleTimeUpdate}
          onCanPlay={handleCanPlay}
          onPause={onPause}
          className="h-full w-full [&_video]:h-full [&_video]:w-full [&_video]:object-contain"
        >
          <MediaProvider className="h-full w-full" />
          {controls}
        </MediaPlayer>
      </div>
    </div>
  )

  return (
    <>
      <div ref={slotRef} className="pointer-events-none mb-4 aspect-video w-full rounded-lg bg-black" />
      {createPortal(playerLayer, document.body)}
    </>
  )
}

function useSlotRect(ref: React.RefObject<HTMLElement | null>, active: boolean) {
  const [rect, setRect] = useState<{ top: number; left: number; width: number; height: number } | null>(null)

  useEffect(() => {
    if (!active) return
    const el = ref.current
    if (!el) return
    const measure = () => {
      const r = el.getBoundingClientRect()
      setRect({ top: r.top, left: r.left, width: r.width, height: r.height })
    }
    measure()
    const observer = new ResizeObserver(measure)
    observer.observe(el)
    window.addEventListener("scroll", measure, true)
    window.addEventListener("resize", measure)
    return () => {
      observer.disconnect()
      window.removeEventListener("scroll", measure, true)
      window.removeEventListener("resize", measure)
    }
  }, [ref, active])

  return rect
}
