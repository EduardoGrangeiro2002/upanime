import { useState } from "react"
import { ChevronDown, ChevronUp, Trash2 } from "lucide-react"
import type { Download } from "@/api/types"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { DownloadItem } from "./download-item"

interface DownloadAnimeCardProps {
  animeId: string
  animeTitle: string
  animeImageUrl: string
  downloads: Download[]
  onCancel: (id: string) => void
  onClearCompleted: (animeId: string) => void
}

export function DownloadAnimeCard({
  animeTitle,
  animeImageUrl,
  downloads,
  animeId,
  onCancel,
  onClearCompleted,
}: DownloadAnimeCardProps) {
  const [expanded, setExpanded] = useState(true)

  const completedCount = downloads.filter((d) => d.status === "completed").length
  const totalCount = downloads.length
  const activeCount = downloads.filter((d) => d.status === "downloading" || d.status === "resolving").length
  const completedPct = totalCount > 0 ? Math.round((completedCount / totalCount) * 100) : 0
  const hasCompleted = completedCount > 0

  return (
    <Card className="glass rounded-xl">
      <button
        onClick={() => setExpanded((p) => !p)}
        aria-expanded={expanded}
        className="flex w-full items-center gap-3 p-3 text-left transition-colors hover:bg-surface-high/50 rounded-t-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        {animeImageUrl && (
          <img
            src={animeImageUrl}
            alt=""
            className="h-14 w-10 shrink-0 rounded-lg object-cover"
          />
        )}
        <div className="flex-1 min-w-0">
          <h4 className="text-sm font-semibold truncate" title={animeTitle}>{animeTitle}</h4>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <span>{completedCount}/{totalCount} completos</span>
            {activeCount > 0 && <span className="text-primary">{activeCount} baixando</span>}
          </div>
          <div className="mt-1 h-1 w-full overflow-hidden rounded-full bg-surface-high">
            <div
              className="h-full rounded-full bg-gradient-to-r from-primary to-primary-dim transition-all duration-300"
              style={{ width: `${completedPct}%` }}
            />
          </div>
        </div>
        {expanded ? <ChevronUp className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" /> : <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />}
      </button>

      {expanded && (
        <CardContent className="px-3 pb-3 pt-0">
          {hasCompleted && (
            <div className="flex justify-end mb-2">
              <Button variant="ghost" size="sm" onClick={() => onClearCompleted(animeId)} className="h-7 text-xs">
                <Trash2 className="h-3 w-3" />
                Limpar completos
              </Button>
            </div>
          )}
          <div className="max-h-[400px] space-y-2 overflow-y-auto scrollbar-thin pr-1">
            {downloads.map((dl) => (
              <DownloadItem key={dl.id} download={dl} onCancel={onCancel} />
            ))}
          </div>
        </CardContent>
      )}
    </Card>
  )
}
