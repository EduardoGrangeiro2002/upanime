import { useEffect, useRef, useCallback, useState } from "react"
import { createPortal } from "react-dom"
import { MediaPlayer, MediaProvider, type MediaPlayerInstance } from "@vidstack/react"
import { PlayerControls } from "./player-controls"
import { usePseudoFullscreen } from "@/hooks/use-fullscreen"
import { pseudoFullscreenStyle, needsCssRotation } from "@/lib/fullscreen"
import { buildQualityOptions } from "@/lib/quality"
import type { Episode } from "@/api/types"

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
  const { isFullscreen, toggle, usesRealFullscreen } = usePseudoFullscreen(episode.id)

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

  const controls = (
    <PlayerControls
      onClose={onClose}
      onPrevious={onPrevious}
      onNext={onNext}
      qualities={qualities}
      activeQuality={activeQuality}
      onSelectQuality={selectQuality}
      isFullscreen={isFullscreen}
      onToggleFullscreen={toggle}
      menuOpen={menuOpen}
      onMenuOpenChange={setMenuOpen}
    />
  )

  const player = (
    <MediaPlayer
      ref={playerRef}
      key={episode.id}
      src={{ src: source, type: "video/mp4" }}
      title={title}
      aspectRatio="16/9"
      playsInline
      autoPlay={autoPlay}
      fullscreenOrientation="landscape"
      onTimeUpdate={handleTimeUpdate}
      onCanPlay={handleCanPlay}
      onPause={onPause}
      className="h-full w-full"
    >
      <MediaProvider />
      {controls}
    </MediaPlayer>
  )

  if (isFullscreen && !usesRealFullscreen) {
    return createPortal(
      <div style={pseudoFullscreenStyle(needsCssRotation())}>{player}</div>,
      document.body,
    )
  }

  return <div className="relative overflow-hidden rounded-lg bg-black mb-4">{player}</div>
}
