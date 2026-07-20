import { useEffect, useRef, useCallback } from "react"
import { MediaPlayer, MediaProvider, type MediaPlayerInstance } from "@vidstack/react"
import { DefaultVideoLayout, defaultLayoutIcons } from "@vidstack/react/player/layouts/default"
import { Button } from "@/components/ui/button"
import { X, SkipBack, SkipForward } from "lucide-react"

type NativeFullscreenVideo = HTMLVideoElement & { webkitEnterFullscreen: () => void }

function supportsNativeFullscreen(video: HTMLVideoElement | null): video is NativeFullscreenVideo {
  return !!video && typeof (video as Partial<NativeFullscreenVideo>).webkitEnterFullscreen === "function"
}

export function enterNativeFullscreen(video: HTMLVideoElement | null, event: Event) {
  if (!supportsNativeFullscreen(video)) return false
  event.preventDefault()
  event.stopImmediatePropagation()
  video.webkitEnterFullscreen()
  return true
}

interface VideoPlayerProps {
  src: string
  title: string
  episodeId: string
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
  episodeId,
  autoPlay = false,
  onClose,
  onPrevious,
  onNext,
  initialTime,
  onTimeUpdate,
  onPause,
}: VideoPlayerProps) {
  const playerRef = useRef<MediaPlayerInstance>(null)
  const wrapperRef = useRef<HTMLDivElement>(null)
  const seekedRef = useRef(false)

  useEffect(() => {
    const wrapper = wrapperRef.current
    if (!wrapper) return
    const handleRequest = (event: Event) => {
      enterNativeFullscreen(wrapper.querySelector("video"), event)
    }
    wrapper.addEventListener("media-enter-fullscreen-request", handleRequest, true)
    return () => wrapper.removeEventListener("media-enter-fullscreen-request", handleRequest, true)
  }, [])

  const handleTimeUpdate = useCallback(() => {
    const player = playerRef.current
    if (!player) return
    onTimeUpdate?.(player.currentTime, player.duration)
  }, [onTimeUpdate])

  useEffect(() => {
    seekedRef.current = false
  }, [episodeId])

  const handleCanPlay = useCallback(() => {
    if (seekedRef.current || !initialTime || initialTime < 2) return
    const player = playerRef.current
    if (!player) return
    player.currentTime = initialTime
    seekedRef.current = true
  }, [initialTime])

  return (
    <div ref={wrapperRef} className="relative rounded-lg overflow-hidden bg-black mb-4">
      <div className="absolute top-2 right-2 z-50 flex items-center gap-1">
        {onPrevious && (
          <Button
            variant="ghost"
            size="icon"
            className="h-10 w-10 md:h-8 md:w-8 bg-black/60 hover:bg-black/80 text-white"
            onClick={onPrevious}
            aria-label="Episódio anterior"
            data-tooltip="Episódio anterior"
          >
            <SkipBack className="h-4 w-4" aria-hidden="true" />
          </Button>
        )}
        {onNext && (
          <Button
            variant="ghost"
            size="icon"
            className="h-10 w-10 md:h-8 md:w-8 bg-black/60 hover:bg-black/80 text-white"
            onClick={onNext}
            aria-label="Próximo episódio"
            data-tooltip="Próximo episódio"
          >
            <SkipForward className="h-4 w-4" aria-hidden="true" />
          </Button>
        )}
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 bg-black/60 hover:bg-black/80 text-white"
          onClick={onClose}
          aria-label="Fechar player"
          data-tooltip="Fechar player"
          data-tooltip-pos="left"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </Button>
      </div>
      <MediaPlayer
        ref={playerRef}
        key={src}
        src={{ src, type: "video/mp4" }}
        title={title}
        aspectRatio="16/9"
        playsInline
        autoPlay={autoPlay}
        onTimeUpdate={handleTimeUpdate}
        onCanPlay={handleCanPlay}
        onPause={onPause}
      >
        <MediaProvider />
        <DefaultVideoLayout icons={defaultLayoutIcons} colorScheme="dark" />
      </MediaPlayer>
    </div>
  )
}
