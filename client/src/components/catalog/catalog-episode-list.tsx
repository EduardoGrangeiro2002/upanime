import { useState, useRef, useEffect, useMemo } from "react"
import type { Episode, EpisodeStreamVariant, Season, WatchProgressItem } from "@/api/types"
import { Button } from "@/components/ui/button"
import { Play, Sparkles, Trash2 } from "lucide-react"
import { episodeThumbnailURL } from "@/api/endpoints"
import { useEpisodeStream } from "@/hooks/use-catalog"
import { usePlaybackProgress, useWatchProgressList, buildProgressMap, progressPct } from "@/hooks/use-playback-progress"
import { VideoPlayer } from "./video-player"

interface CatalogEpisodeListProps {
  animeTitle: string
  season: Season
  autoPlayOnOpen?: boolean
  onDeleteEpisode: (id: string) => void
  onDeleteUpscaledEpisode: (id: string) => void
  isDeleting: boolean
  isDeletingUpscaled: boolean
}

function episodeLabel(number: string, type: string, index: number): string {
  const num = number || String(index + 1)
  if (type === "movie") return "Filme"
  if (type === "ova") return `OVA ${num}`
  return `Episódio ${num.padStart(2, "0")}`
}

function pickInitialEpisode(episodes: Episode[], progressMap: Record<string, WatchProgressItem>): string | null {
  const inProgress = episodes.find((ep) => (progressMap[ep.id]?.position ?? 0) > 0)
  if (inProgress) return inProgress.id
  return episodes[0]?.id ?? null
}

function formatTime(seconds: number): string {
  return `${Math.floor(seconds / 60)}:${String(Math.floor(seconds % 60)).padStart(2, "0")}`
}

export function CatalogEpisodeList({
  animeTitle,
  season,
  autoPlayOnOpen = false,
  onDeleteEpisode,
  onDeleteUpscaledEpisode,
  isDeleting,
  isDeletingUpscaled,
}: CatalogEpisodeListProps) {
  const downloaded = season.episodes.filter((ep) => ep.storageKey)
  const [confirmKey, setConfirmKey] = useState<string | null>(null)
  const [activeEpisodeId, setActiveEpisodeId] = useState<string | null>(null)
  const [activeVariant, setActiveVariant] = useState<EpisodeStreamVariant>("original")
  const { data: progressList, isFetched: progressFetched } = useWatchProgressList()
  const progressMap = useMemo(() => buildProgressMap(progressList), [progressList])
  const autoPickedRef = useRef(false)
  const { data: streamData } = useEpisodeStream(activeEpisodeId, activeVariant)
  const { savedTime, ready, handleTimeUpdate, flush } = usePlaybackProgress(activeEpisodeId)
  const listRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!autoPlayOnOpen || autoPickedRef.current || !progressFetched) return
    autoPickedRef.current = true
    setActiveEpisodeId(pickInitialEpisode(downloaded, progressMap))
  }, [autoPlayOnOpen, progressFetched, downloaded, progressMap])

  const activeIndex = activeEpisodeId
    ? downloaded.findIndex((ep) => ep.id === activeEpisodeId)
    : -1
  const activeEpisode = activeIndex >= 0 ? downloaded[activeIndex] : null

  const activeTitle = activeEpisode
    ? `${animeTitle} — ${episodeLabel(activeEpisode.number, activeEpisode.type, activeIndex)}`
    : ""

  const hasPrevious = activeIndex > 0
  const hasNext = activeIndex >= 0 && activeIndex < downloaded.length - 1

  const goToEpisode = (index: number) => {
    const ep = downloaded[index]
    if (!ep) return
    setActiveEpisodeId(ep.id)
    setActiveVariant("original")
  }

  useEffect(() => {
    if (activeIndex < 0 || !listRef.current) return
    const item = listRef.current.children[activeIndex] as HTMLElement | undefined
    item?.scrollIntoView?.({ behavior: "smooth", block: "nearest" })
  }, [activeIndex])

  return (
    <div className="space-y-3">
      {activeEpisode && streamData?.url && ready && (
        <div className="space-y-2">
          {activeEpisode.upscaledStorageKey && (
            <div className="flex items-center gap-2" role="group" aria-label="Versão do vídeo">
              <Button
                size="sm"
                variant={activeVariant === "original" ? "default" : "outline"}
                onClick={() => setActiveVariant("original")}
              >
                Original
              </Button>
              <Button
                size="sm"
                variant={activeVariant === "upscaled" ? "default" : "outline"}
                onClick={() => setActiveVariant("upscaled")}
              >
                Upscale
              </Button>
            </div>
          )}
          <VideoPlayer
            src={streamData.url}
            title={activeTitle}
            episodeId={activeEpisode.id}
            autoPlay
            onClose={() => setActiveEpisodeId(null)}
            onPrevious={hasPrevious ? () => goToEpisode(activeIndex - 1) : undefined}
            onNext={hasNext ? () => goToEpisode(activeIndex + 1) : undefined}
            initialTime={savedTime}
            onTimeUpdate={handleTimeUpdate}
            onPause={flush}
          />
        </div>
      )}

      <h3 className="text-sm font-medium text-muted-foreground">
        Episódios · {downloaded.length}
      </h3>

      <div
        ref={listRef}
        className="space-y-1 max-h-[240px] md:max-h-[360px] overflow-y-auto scrollbar-thin pr-1"
      >
        {downloaded.map((ep, index) => {
          const label = episodeLabel(ep.number, ep.type, index)
          const entry = progressMap[ep.id]
          const progressSeconds = entry?.position ?? 0
          const pct = entry ? progressPct(entry.position, entry.duration) : 0
          const isActive = ep.id === activeEpisodeId

          return (
            <div
              key={ep.id}
              className={`group/ep flex items-center gap-3 rounded-lg px-3 py-3 transition-colors ${
                isActive
                  ? "bg-primary/10 ring-1 ring-primary/30"
                  : "hover:bg-surface-high"
              }`}
            >
              <span className="font-display text-base font-medium text-muted-foreground w-6 text-center shrink-0">
                {index + 1}
              </span>

              <button
                onClick={() => {
                  setActiveEpisodeId(ep.id)
                  setActiveVariant("original")
                }}
                aria-label={`Assistir ${label}`}
                className="relative shrink-0 h-[56px] w-[100px] rounded-lg bg-surface-high overflow-hidden group/play cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <img
                  src={episodeThumbnailURL(ep.id)}
                  alt=""
                  loading="lazy"
                  className="absolute inset-0 h-full w-full object-cover"
                  onError={(event) => {
                    event.currentTarget.style.display = "none"
                  }}
                />
                <div className="absolute inset-0 flex items-center justify-center">
                  <div className="h-8 w-8 rounded-full bg-black/60 border border-white/20 flex items-center justify-center group-hover/play:bg-white/20 transition-colors">
                    <Play className="h-3.5 w-3.5 fill-white text-white ml-0.5" aria-hidden="true" />
                  </div>
                </div>
                {pct > 0 && (
                  <div className="absolute bottom-0 left-0 right-0 h-[3px] bg-white/20">
                    <div
                      className="h-full bg-primary"
                      style={{ width: `${pct}%` }}
                      data-testid={`progress-${ep.id}`}
                    />
                  </div>
                )}
              </button>

              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate" title={ep.title}>{label}</p>
                {progressSeconds > 0 && (
                  <p className="text-xs text-muted-foreground">
                    Parou em {formatTime(progressSeconds)}
                  </p>
                )}
              </div>

              <div className="flex items-center gap-1 shrink-0 md:opacity-0 md:group-hover/ep:opacity-100 md:group-focus-within/ep:opacity-100 transition-opacity">
                {ep.upscaledStorageKey && confirmKey !== `ep-${ep.id}` && (
                  confirmKey === `up-${ep.id}` ? (
                    <div className="flex gap-1">
                      <Button
                        variant="destructive"
                        size="sm"
                        aria-label="Remover upscale"
                        className="h-7 text-xs"
                        disabled={isDeletingUpscaled}
                        onClick={() => {
                          onDeleteUpscaledEpisode(ep.id)
                          setConfirmKey(null)
                        }}
                      >
                        Remover<span className="hidden sm:inline"> upscale</span>
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 text-xs"
                        onClick={() => setConfirmKey(null)}
                      >
                        Não
                      </Button>
                    </div>
                  ) : (
                    <Button
                      variant="outline"
                      size="sm"
                      aria-label="Excluir upscale"
                      data-tooltip="Excluir upscale"
                      data-tooltip-pos="left"
                      className="h-7 text-xs"
                      disabled={isDeletingUpscaled}
                      onClick={() => setConfirmKey(`up-${ep.id}`)}
                    >
                      <Sparkles className="h-3.5 w-3.5 sm:mr-1" aria-hidden="true" />
                      <span className="hidden sm:inline">Excluir upscale</span>
                    </Button>
                  )
                )}
                {confirmKey !== `up-${ep.id}` && (confirmKey === `ep-${ep.id}` ? (
                  <div className="flex gap-1">
                    <Button
                      variant="destructive"
                      size="sm"
                      className="h-7 text-xs"
                      disabled={isDeleting}
                      onClick={() => {
                        onDeleteEpisode(ep.id)
                        setConfirmKey(null)
                      }}
                    >
                      Remover
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 text-xs"
                      onClick={() => setConfirmKey(null)}
                    >
                      Não
                    </Button>
                  </div>
                ) : (
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label={`Remover ${label}`}
                    data-tooltip="Remover episódio"
                    data-tooltip-pos="left"
                    className="h-7 w-7 text-muted-foreground hover:text-destructive"
                    onClick={() => setConfirmKey(`ep-${ep.id}`)}
                  >
                    <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                  </Button>
                ))}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
