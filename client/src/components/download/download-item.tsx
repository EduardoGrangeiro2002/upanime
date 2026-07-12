import { X } from "lucide-react"
import type { Download } from "@/api/types"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { ProgressBar } from "./progress-bar"
import { cn } from "@/lib/utils"

interface DownloadItemProps {
  download: Download
  onCancel: (id: string) => void
}

const STATUS_LABELS: Record<string, string> = {
  queued: "Na fila",
  resolving: "Resolvendo",
  downloading: "Baixando",
  completed: "Completo",
  failed: "Falhou",
}

function statusVariant(status: string) {
  if (status === "completed") return "success" as const
  if (status === "failed") return "destructive" as const
  return "secondary" as const
}

export function DownloadItem({ download, onCancel }: DownloadItemProps) {
  const isActive = download.status === "downloading" || download.status === "resolving" || download.status === "queued"

  return (
    <div className={cn("flex items-center gap-3 rounded-xl bg-surface-high/50 p-3", download.status === "completed" && "opacity-60")}>
      <div className="flex-1 min-w-0 space-y-1.5">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium truncate">{download.episodeTitle}</span>
          <Badge variant={statusVariant(download.status)} className="text-[10px] px-1.5 py-0">
            {STATUS_LABELS[download.status] ?? download.status}
          </Badge>
        </div>
        <ProgressBar progress={download.progress} />
        <div className="flex gap-3 text-xs text-muted-foreground tabular-nums">
          <span>{download.progress}%</span>
          {download.speed && <span>{download.speed}</span>}
          {download.eta && <span>ETA: {download.eta}</span>}
          {download.error && <span className="text-destructive">{download.error}</span>}
        </div>
      </div>

      {isActive && (
        <Button
          variant="ghost"
          size="icon"
          onClick={() => onCancel(download.id)}
          aria-label={`Cancelar download de ${download.episodeTitle}`}
          data-tooltip="Cancelar download"
          data-tooltip-pos="left"
          className="shrink-0"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </Button>
      )}
    </div>
  )
}
