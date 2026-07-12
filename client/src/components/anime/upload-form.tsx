import { useRef, useState } from "react"
import { toast } from "sonner"
import { useQueryClient } from "@tanstack/react-query"
import { Upload, Loader2, CheckCircle2, XCircle, FileVideo, Trash2 } from "lucide-react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { ProgressBar } from "@/components/download/progress-bar"
import { uploadEpisode } from "@/api/endpoints"
import { detectEpisodeNumber } from "@/lib/episode-number"
import { cn } from "@/lib/utils"

type FileStatus = "pending" | "uploading" | "done" | "error"

interface QueuedFile {
  file: File
  episodeNumber: string
  status: FileStatus
  progress: number
  error?: string
}

function buildQueue(files: File[]): QueuedFile[] {
  return files.map((file, index) => ({
    file,
    episodeNumber: detectEpisodeNumber(file.name, index + 1),
    status: "pending" as const,
    progress: 0,
  }))
}

export function UploadForm() {
  const [animeTitle, setAnimeTitle] = useState("")
  const [seasonNumber, setSeasonNumber] = useState("1")
  const [queue, setQueue] = useState<QueuedFile[]>([])
  const [isUploading, setIsUploading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const queryClient = useQueryClient()

  const updateItem = (index: number, patch: Partial<QueuedFile>) => {
    setQueue((prev) => prev.map((item, i) => (i === index ? { ...item, ...patch } : item)))
  }

  const handleFiles = (files: FileList | null) => {
    if (!files || files.length === 0) return
    const items = buildQueue(Array.from(files))
    setQueue((prev) => [...prev, ...items])
  }

  const removeItem = (index: number) => {
    setQueue((prev) => prev.filter((_, i) => i !== index))
  }

  const pendingCount = queue.filter((q) => q.status === "pending" || q.status === "error").length
  const canSubmit = animeTitle.trim() !== "" && pendingCount > 0 && !isUploading

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!canSubmit) return

    const season = parseInt(seasonNumber, 10)
    if (Number.isNaN(season) || season < 0) {
      toast.error("Temporada inválida")
      return
    }

    setIsUploading(true)
    let uploaded = 0
    for (let i = 0; i < queue.length; i++) {
      const item = queue[i]
      if (item.status === "done" || item.status === "uploading") continue

      updateItem(i, { status: "uploading", progress: 0, error: undefined })
      try {
        await uploadEpisode(
          {
            animeTitle: animeTitle.trim(),
            seasonNumber: season,
            episodeNumber: item.episodeNumber,
            file: item.file,
          },
          (pct) => updateItem(i, { progress: pct }),
        )
        updateItem(i, { status: "done", progress: 100 })
        uploaded++
      } catch (error) {
        const message = error instanceof Error ? error.message : "erro desconhecido"
        updateItem(i, { status: "error", error: message })
      }
    }
    setIsUploading(false)

    if (uploaded > 0) {
      queryClient.invalidateQueries({ queryKey: ["catalog"] })
      toast.success(
        uploaded === 1
          ? "1 episódio enviado para o catálogo"
          : `${uploaded} episódios enviados para o catálogo`,
      )
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-[1fr_140px] gap-2">
        <Input
          value={animeTitle}
          onChange={(e) => setAnimeTitle(e.target.value)}
          placeholder="Nome do anime"
          aria-label="Nome do anime"
          className="glass"
          disabled={isUploading}
        />
        <Input
          value={seasonNumber}
          onChange={(e) => setSeasonNumber(e.target.value)}
          type="number"
          min={0}
          placeholder="Temporada"
          aria-label="Temporada"
          className="glass"
          disabled={isUploading}
        />
      </div>

      <input
        ref={inputRef}
        type="file"
        accept="video/mp4,video/webm,video/x-matroska,.mp4,.webm,.mkv,.m4v,.mov"
        multiple
        className="hidden"
        onChange={(e) => {
          handleFiles(e.target.files)
          e.target.value = ""
        }}
      />

      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault()
          handleFiles(e.dataTransfer.files)
        }}
        disabled={isUploading}
        className="w-full rounded-xl border-2 border-dashed border-muted-foreground/20 hover:border-muted-foreground/40 p-6 text-center transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
      >
        <span className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
          <FileVideo className="h-4 w-4" aria-hidden="true" />
          Arraste os episódios ou clique para escolher (mp4, webm, mkv)
        </span>
      </button>

      {queue.length > 0 && (
        <div className="space-y-1 max-h-72 overflow-y-auto scrollbar-thin pr-1">
          {queue.map((item, index) => (
            <div
              key={`${item.file.name}-${index}`}
              className={cn(
                "flex items-center gap-3 rounded-lg bg-surface-high/50 px-3 py-2",
                item.status === "done" && "opacity-60",
              )}
            >
              <span className="text-xs text-muted-foreground tabular-nums w-12 shrink-0">
                Ep {item.episodeNumber}
              </span>
              <div className="flex-1 min-w-0">
                <p className="text-sm truncate" title={item.file.name}>{item.file.name}</p>
                {item.status === "uploading" && <ProgressBar progress={item.progress} className="mt-1 h-1" />}
                {item.status === "error" && (
                  <p className="text-xs text-destructive">{item.error}</p>
                )}
              </div>
              {item.status === "pending" && (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label={`Remover ${item.file.name} da lista`}
                  data-tooltip="Remover da lista"
                  data-tooltip-pos="left"
                  className="h-7 w-7 shrink-0 text-muted-foreground"
                  onClick={() => removeItem(index)}
                >
                  <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                </Button>
              )}
              {item.status === "uploading" && <Loader2 className="h-4 w-4 shrink-0 animate-spin text-primary" aria-hidden="true" />}
              {item.status === "done" && <CheckCircle2 className="h-4 w-4 shrink-0 text-success" aria-hidden="true" />}
              {item.status === "error" && <XCircle className="h-4 w-4 shrink-0 text-destructive" aria-hidden="true" />}
            </div>
          ))}
        </div>
      )}

      <Button type="submit" variant="gradient" disabled={!canSubmit}>
        {isUploading ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Upload className="h-4 w-4" aria-hidden="true" />}
        {pendingCount > 0 ? `Enviar ${pendingCount} episódio${pendingCount !== 1 ? "s" : ""}` : "Enviar episódios"}
      </Button>
    </form>
  )
}
