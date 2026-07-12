import { useRef, useState, useCallback } from "react"
import { Play, Pause, StepBack, StepForward, Upload, Volume2, VolumeX, Library } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useCatalog } from "@/hooks/use-catalog"
import { fetchEpisodeStreamURL } from "@/api/endpoints"
import type { Anime, Episode } from "@/api/types"

const FRAME_SECONDS = 1 / 24

interface LoadedVideo {
  url: string
  name: string
}

type CompareSource = "catalog" | "upload"

function revokeIfBlob(video: LoadedVideo | null) {
  if (video && video.url.startsWith("blob:")) URL.revokeObjectURL(video.url)
}

export function ComparePage() {
  const [source, setSource] = useState<CompareSource>("catalog")
  const [videoA, setVideoA] = useState<LoadedVideo | null>(null)
  const [videoB, setVideoB] = useState<LoadedVideo | null>(null)
  const refA = useRef<HTMLVideoElement>(null)
  const refB = useRef<HTMLVideoElement>(null)
  const [playing, setPlaying] = useState(false)
  const [muteA, setMuteA] = useState(false)
  const [muteB, setMuteB] = useState(true)
  const [duration, setDuration] = useState(0)
  const [currentTime, setCurrentTime] = useState(0)

  const loadedRefs = useCallback(() => {
    return [refA.current, refB.current].filter((v): v is HTMLVideoElement => v !== null)
  }, [])

  const resetPlayback = useCallback(() => {
    setPlaying(false)
    setDuration(0)
    setCurrentTime(0)
  }, [])

  const handleFile = useCallback((side: "a" | "b", file: File) => {
    const loaded = { url: URL.createObjectURL(file), name: file.name }
    if (side === "a") {
      revokeIfBlob(videoA)
      setVideoA(loaded)
      return
    }
    revokeIfBlob(videoB)
    setVideoB(loaded)
  }, [videoA, videoB])

  const handleCatalogEpisode = useCallback(async (episode: Episode) => {
    const [original, upscaled] = await Promise.all([
      fetchEpisodeStreamURL(episode.id, "original"),
      fetchEpisodeStreamURL(episode.id, "upscaled"),
    ])
    revokeIfBlob(videoA)
    revokeIfBlob(videoB)
    setVideoA({ url: original.url, name: `${episode.title} (original)` })
    setVideoB({ url: upscaled.url, name: `${episode.title} (upscale)` })
    resetPlayback()
  }, [videoA, videoB, resetPlayback])

  const updateDuration = useCallback(() => {
    const durations = loadedRefs().map((v) => v.duration).filter((d) => Number.isFinite(d))
    setDuration(durations.length > 0 ? Math.max(...durations) : 0)
  }, [loadedRefs])

  const seek = useCallback((time: number) => {
    for (const video of loadedRefs()) {
      video.currentTime = time
    }
    setCurrentTime(time)
  }, [loadedRefs])

  const togglePlayback = useCallback(() => {
    const videos = loadedRefs()
    if (videos.length === 0) return
    if (playing) {
      videos.forEach((v) => v.pause())
      setPlaying(false)
      return
    }
    videos.forEach((v) => { v.play() })
    setPlaying(true)
  }, [playing, loadedRefs])

  const stepFrame = useCallback((direction: 1 | -1) => {
    loadedRefs().forEach((v) => v.pause())
    setPlaying(false)
    const next = Math.min(Math.max(currentTime + direction * FRAME_SECONDS, 0), duration || 0)
    seek(next)
  }, [currentTime, duration, seek, loadedRefs])

  const hasVideo = videoA !== null || videoB !== null

  return (
    <div className="px-4 py-4 md:px-8 md:py-8 max-w-7xl mx-auto space-y-6">
      <div>
        <h1 className="font-display text-2xl md:text-3xl font-bold tracking-tighter">Comparar Vídeos</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Compare o original com a versão upscale de um episódio do catálogo, ou envie dois arquivos seus.
        </p>
      </div>

      <div className="flex gap-2" role="group" aria-label="Origem dos vídeos">
        <Button
          type="button"
          size="sm"
          variant={source === "catalog" ? "default" : "outline"}
          aria-pressed={source === "catalog"}
          onClick={() => setSource("catalog")}
        >
          <Library className="h-4 w-4" aria-hidden="true" />
          Do catálogo
        </Button>
        <Button
          type="button"
          size="sm"
          variant={source === "upload" ? "default" : "outline"}
          aria-pressed={source === "upload"}
          onClick={() => setSource("upload")}
        >
          <Upload className="h-4 w-4" aria-hidden="true" />
          Upload manual
        </Button>
      </div>

      {source === "catalog" ? (
        <CatalogPicker onSelect={handleCatalogEpisode} />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <DropZone label="Vídeo A (Original)" fileName={videoA?.name} onFile={(f) => handleFile("a", f)} />
          <DropZone label="Vídeo B (Processado)" fileName={videoB?.name} onFile={(f) => handleFile("b", f)} />
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <VideoPane
          label="Vídeo A"
          video={videoA}
          videoRef={refA}
          muted={muteA}
          onToggleMute={() => setMuteA((m) => !m)}
          onLoadedMetadata={updateDuration}
          onTimeUpdate={(t) => setCurrentTime(t)}
          onPause={() => setPlaying(false)}
        />
        <VideoPane
          label="Vídeo B"
          video={videoB}
          videoRef={refB}
          muted={muteB}
          onToggleMute={() => setMuteB((m) => !m)}
          onLoadedMetadata={updateDuration}
          onTimeUpdate={videoA ? undefined : (t) => setCurrentTime(t)}
          onPause={() => setPlaying(false)}
        />
      </div>

      {hasVideo && (
        <div className="rounded-xl glass border border-white/[0.06] p-4 space-y-3">
          <input
            type="range"
            min={0}
            max={duration || 0}
            step={FRAME_SECONDS}
            value={Math.min(currentTime, duration || 0)}
            disabled={duration === 0}
            aria-label="Linha do tempo"
            onChange={(e) => seek(parseFloat(e.target.value))}
            className="w-full h-1.5 rounded-full appearance-none bg-surface-high accent-primary cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
          <div className="flex items-center justify-center gap-2">
            <span className="text-xs text-muted-foreground tabular-nums w-24">
              {formatTime(currentTime)} / {formatTime(duration)}
            </span>
            <Button
              variant="outline"
              size="icon"
              aria-label="Frame anterior"
              data-tooltip="Frame anterior"
              onClick={() => stepFrame(-1)}
              disabled={duration === 0}
            >
              <StepBack className="h-4 w-4" aria-hidden="true" />
            </Button>
            <Button variant="gradient" onClick={togglePlayback} className="w-28">
              {playing ? <Pause className="h-4 w-4" aria-hidden="true" /> : <Play className="h-4 w-4" aria-hidden="true" />}
              {playing ? "Pausar" : "Play"}
            </Button>
            <Button
              variant="outline"
              size="icon"
              aria-label="Próximo frame"
              data-tooltip="Próximo frame"
              onClick={() => stepFrame(1)}
              disabled={duration === 0}
            >
              <StepForward className="h-4 w-4" aria-hidden="true" />
            </Button>
            <span className="w-24" aria-hidden="true" />
          </div>
        </div>
      )}
    </div>
  )
}

function formatTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return "0:00"
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${String(s).padStart(2, "0")}`
}

function VideoPane({
  label,
  video,
  videoRef,
  muted,
  onToggleMute,
  onLoadedMetadata,
  onTimeUpdate,
  onPause,
}: {
  label: string
  video: LoadedVideo | null
  videoRef: React.RefObject<HTMLVideoElement | null>
  muted: boolean
  onToggleMute: () => void
  onLoadedMetadata: () => void
  onTimeUpdate?: (time: number) => void
  onPause: () => void
}) {
  return (
    <div className="space-y-2">
      <div className="rounded-xl bg-surface overflow-hidden aspect-video">
        {video ? (
          <video
            ref={videoRef}
            src={video.url}
            muted={muted}
            className="w-full h-full object-contain"
            onLoadedMetadata={onLoadedMetadata}
            onTimeUpdate={onTimeUpdate ? (e) => onTimeUpdate(e.currentTarget.currentTime) : undefined}
            onPause={onPause}
          />
        ) : (
          <div className="flex items-center justify-center h-full text-muted-foreground text-sm">{label}</div>
        )}
      </div>
      {video && (
        <Button
          variant="outline"
          size="sm"
          onClick={onToggleMute}
          aria-pressed={!muted}
          className="text-xs"
        >
          {muted ? <VolumeX className="h-3.5 w-3.5" aria-hidden="true" /> : <Volume2 className="h-3.5 w-3.5" aria-hidden="true" />}
          {muted ? `Som ${label.slice(-1)}: mudo` : `Som ${label.slice(-1)}: ativo`}
        </Button>
      )}
    </div>
  )
}

function upscaledEpisodes(anime: Anime): Episode[] {
  return anime.seasons.flatMap((season) =>
    season.episodes.filter((ep) => ep.storageKey && ep.upscaledStorageKey),
  )
}

function CatalogPicker({ onSelect }: { onSelect: (episode: Episode) => void }) {
  const { data: animes, isLoading } = useCatalog()
  const [animeId, setAnimeId] = useState("")
  const [episodeId, setEpisodeId] = useState("")

  const withUpscale = (animes ?? []).filter((anime) => upscaledEpisodes(anime).length > 0)
  const selectedAnime = withUpscale.find((anime) => anime.id === animeId)
  const episodes = selectedAnime ? upscaledEpisodes(selectedAnime) : []

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">Carregando catálogo…</p>
  }

  if (withUpscale.length === 0) {
    return (
      <div className="rounded-xl bg-surface p-6 text-center">
        <p className="text-sm text-muted-foreground">
          Nenhum episódio com versão upscale no catálogo. Rode um upscale primeiro ou use o upload manual.
        </p>
      </div>
    )
  }

  const selectClass =
    "h-9 rounded-lg bg-input px-3 text-sm border border-border focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
      <select
        value={animeId}
        aria-label="Anime"
        className={selectClass}
        onChange={(e) => {
          setAnimeId(e.target.value)
          setEpisodeId("")
        }}
      >
        <option value="">Escolha um anime…</option>
        {withUpscale.map((anime) => (
          <option key={anime.id} value={anime.id}>{anime.title}</option>
        ))}
      </select>

      <select
        value={episodeId}
        aria-label="Episódio"
        className={selectClass}
        disabled={!selectedAnime}
        onChange={(e) => {
          setEpisodeId(e.target.value)
          const episode = episodes.find((ep) => ep.id === e.target.value)
          if (episode) onSelect(episode)
        }}
      >
        <option value="">Escolha um episódio…</option>
        {episodes.map((ep) => (
          <option key={ep.id} value={ep.id}>
            {ep.seasonNumber}x{ep.number.padStart(2, "0")} — {ep.title}
          </option>
        ))}
      </select>
    </div>
  )
}

function DropZone({ label, fileName, onFile }: { label: string; fileName?: string; onFile: (file: File) => void }) {
  const [dragging, setDragging] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  return (
    <div className="relative">
      <input
        ref={inputRef}
        type="file"
        accept="video/*"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0]
          if (file) onFile(file)
          e.target.value = ""
        }}
      />
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        onDragOver={(e) => { e.preventDefault(); setDragging(true) }}
        onDragLeave={() => setDragging(false)}
        onDrop={(e) => {
          e.preventDefault()
          setDragging(false)
          const file = e.dataTransfer.files[0]
          if (file) onFile(file)
        }}
        className={`w-full rounded-xl border-2 border-dashed p-4 text-center cursor-pointer transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
          dragging ? "border-primary bg-primary/10" : "border-muted-foreground/20 hover:border-muted-foreground/40"
        }`}
      >
        <span className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
          <Upload className="h-4 w-4 shrink-0" aria-hidden="true" />
          {fileName ? (
            <span className="truncate max-w-full" title={fileName}>{label}: {fileName}</span>
          ) : (
            <span>{label} — arraste um arquivo ou clique para escolher</span>
          )}
        </span>
      </button>
    </div>
  )
}
