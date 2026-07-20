import { useState, useEffect, useRef } from "react"
import { PlayButton, MuteButton, SeekButton, TimeSlider, VolumeSlider, Time, useMediaState } from "@vidstack/react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { X, SkipBack, SkipForward, Play, Pause, Volume2, VolumeX, RotateCcw, RotateCw, Settings, Maximize, Minimize } from "lucide-react"
import type { QualityOption } from "@/lib/quality"

interface PlayerControlsProps {
  onClose: () => void
  onPrevious?: () => void
  onNext?: () => void
  qualities: QualityOption[]
  activeQuality: string
  onSelectQuality: (variant: string) => void
  isFullscreen: boolean
  onToggleFullscreen: () => void
  menuOpen: boolean
  onMenuOpenChange: (open: boolean) => void
}

const barButton = "flex h-10 w-10 shrink-0 items-center justify-center rounded-md text-white transition hover:bg-white/10"

export function PlayerControls({
  onClose,
  onPrevious,
  onNext,
  qualities,
  activeQuality,
  onSelectQuality,
  isFullscreen,
  onToggleFullscreen,
  menuOpen,
  onMenuOpenChange,
}: PlayerControlsProps) {
  const paused = useMediaState("paused")
  const muted = useMediaState("muted")
  const playing = useMediaState("playing")
  const visible = useAutoHide(playing && !menuOpen)

  return (
    <>
      <div
        className={cn(
          "absolute inset-0 z-40 flex flex-col justify-end transition-opacity",
          visible ? "opacity-100" : "opacity-0",
        )}
      >
        <div className="absolute top-2 right-2">
          <Button
            variant="ghost"
            size="icon"
            className="h-10 w-10 bg-black/60 text-white hover:bg-black/80"
            onClick={onClose}
            aria-label="Fechar player"
            data-tooltip="Fechar player"
            data-tooltip-pos="left"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </Button>
        </div>

        <div className="flex flex-1 items-center justify-center">
          <PlayButton
            className="flex h-16 w-16 items-center justify-center rounded-full bg-black/50 text-white transition hover:bg-primary hover:text-primary-foreground"
            aria-label={paused ? "Reproduzir" : "Pausar"}
          >
            {paused ? <Play className="h-7 w-7" aria-hidden="true" /> : <Pause className="h-7 w-7" aria-hidden="true" />}
          </PlayButton>
        </div>

        <div className="bg-gradient-to-t from-black/90 to-transparent px-3 pb-2 pt-8">
          <TimeSlider.Root className="group relative flex h-6 w-full items-center">
            <TimeSlider.Track className="relative h-1 w-full rounded bg-white/30">
              <TimeSlider.TrackFill className="absolute h-full rounded bg-primary" />
            </TimeSlider.Track>
            <TimeSlider.Thumb className="absolute h-3.5 w-3.5 rounded-full bg-primary" />
          </TimeSlider.Root>

          <div className="flex items-center gap-1 text-white">
            <SeekButton seconds={-10} className={barButton} aria-label="Voltar 10 segundos">
              <RotateCcw className="h-5 w-5" aria-hidden="true" />
            </SeekButton>
            <PlayButton className={barButton} aria-label={paused ? "Reproduzir" : "Pausar"}>
              {paused ? <Play className="h-5 w-5" aria-hidden="true" /> : <Pause className="h-5 w-5" aria-hidden="true" />}
            </PlayButton>
            <SeekButton seconds={10} className={barButton} aria-label="Avançar 10 segundos">
              <RotateCw className="h-5 w-5" aria-hidden="true" />
            </SeekButton>

            <div className="hidden items-center md:flex">
              <MuteButton className={barButton} aria-label={muted ? "Ativar som" : "Silenciar"}>
                {muted ? <VolumeX className="h-5 w-5" aria-hidden="true" /> : <Volume2 className="h-5 w-5" aria-hidden="true" />}
              </MuteButton>
              <VolumeSlider.Root className="group relative flex h-6 w-20 items-center">
                <VolumeSlider.Track className="relative h-1 w-full rounded bg-white/30">
                  <VolumeSlider.TrackFill className="absolute h-full rounded bg-primary" />
                </VolumeSlider.Track>
                <VolumeSlider.Thumb className="absolute h-3 w-3 rounded-full bg-primary opacity-0 transition group-hover:opacity-100" />
              </VolumeSlider.Root>
            </div>

            <div className="ml-2 flex items-center gap-1 text-sm tabular-nums">
              <Time type="current" />
              <span className="text-white/60">/</span>
              <Time type="duration" />
            </div>

            <div className="ml-auto flex items-center gap-1">
              {onPrevious && (
                <button type="button" className={barButton} onClick={onPrevious} aria-label="Episódio anterior">
                  <SkipBack className="h-5 w-5" aria-hidden="true" />
                </button>
              )}
              {onNext && (
                <button type="button" className={barButton} onClick={onNext} aria-label="Próximo episódio">
                  <SkipForward className="h-5 w-5" aria-hidden="true" />
                </button>
              )}
              {qualities.length > 1 && (
                <QualityMenu
                  qualities={qualities}
                  activeQuality={activeQuality}
                  onSelect={onSelectQuality}
                  onOpenChange={onMenuOpenChange}
                />
              )}
              <button
                type="button"
                className={barButton}
                onClick={onToggleFullscreen}
                aria-label={isFullscreen ? "Sair da tela cheia" : "Tela cheia"}
              >
                {isFullscreen ? <Minimize className="h-5 w-5" aria-hidden="true" /> : <Maximize className="h-5 w-5" aria-hidden="true" />}
              </button>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}

interface QualityMenuProps {
  qualities: QualityOption[]
  activeQuality: string
  onSelect: (variant: string) => void
  onOpenChange: (open: boolean) => void
}

function QualityMenu({ qualities, activeQuality, onSelect, onOpenChange }: QualityMenuProps) {
  const [open, setOpen] = useState(false)

  const setMenu = (next: boolean) => {
    setOpen(next)
    onOpenChange(next)
  }

  return (
    <div className="relative">
      <button
        type="button"
        className={barButton}
        onClick={() => setMenu(!open)}
        aria-label="Qualidade"
        aria-haspopup="menu"
        aria-expanded={open}
      >
        <Settings className="h-5 w-5" aria-hidden="true" />
      </button>
      {open && (
        <div role="menu" className="absolute bottom-12 right-0 min-w-32 overflow-hidden rounded-lg bg-surface/95 py-1 shadow-2xl backdrop-blur-md">
          {qualities.map((quality) => (
            <button
              key={quality.variant}
              type="button"
              role="menuitemradio"
              aria-checked={quality.variant === activeQuality}
              className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm text-white hover:bg-primary/20 aria-checked:text-primary"
              onClick={() => {
                onSelect(quality.variant)
                setMenu(false)
              }}
            >
              {quality.label}
              {quality.variant === activeQuality && <span aria-hidden="true">✓</span>}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

function useAutoHide(canHide: boolean): boolean {
  const [visible, setVisible] = useState(true)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (!canHide) {
      setVisible(true)
      return
    }
    const reveal = () => {
      setVisible(true)
      if (timerRef.current) clearTimeout(timerRef.current)
      timerRef.current = setTimeout(() => setVisible(false), 3000)
    }
    reveal()
    window.addEventListener("pointermove", reveal)
    window.addEventListener("pointerdown", reveal)
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
      window.removeEventListener("pointermove", reveal)
      window.removeEventListener("pointerdown", reveal)
    }
  }, [canHide])

  return visible
}
