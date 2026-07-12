import { useMemo } from "react"
import { toast } from "sonner"
import { useDownloads } from "@/hooks/use-downloads"
import { DownloadAnimeCard } from "./download-anime-card"
import { cancelDownload } from "@/api/endpoints"
import type { Download } from "@/api/types"

interface AnimeGroup {
  animeId: string
  animeTitle: string
  animeImageUrl: string
  downloads: Download[]
}

function groupByAnime(downloads: Record<string, Download>): AnimeGroup[] {
  const groups = new Map<string, AnimeGroup>()

  for (const dl of Object.values(downloads)) {
    const existing = groups.get(dl.animeId)
    if (existing) {
      existing.downloads.push(dl)
      continue
    }
    groups.set(dl.animeId, {
      animeId: dl.animeId,
      animeTitle: dl.animeTitle,
      animeImageUrl: dl.animeImageUrl,
      downloads: [dl],
    })
  }

  return Array.from(groups.values())
}

export function DownloadQueue() {
  const downloads = useDownloads((s) => s.downloads)
  const removeDownload = useDownloads((s) => s.removeDownload)
  const clearCompletedForAnime = useDownloads((s) => s.clearCompletedForAnime)

  const groups = useMemo(() => groupByAnime(downloads), [downloads])

  if (groups.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
        <p className="text-sm">Nenhum download em andamento.</p>
        <p className="text-xs mt-1">Cole a URL de um anime para começar.</p>
      </div>
    )
  }

  const handleCancel = async (id: string) => {
    try {
      await cancelDownload(id)
      toast.success("Download cancelado")
    } catch {
      toast.error("Não foi possível cancelar no servidor", {
        description: "O item foi removido da fila local.",
      })
    }
    removeDownload(id)
  }

  return (
    <div className="space-y-4">
      {groups.map((group) => (
        <DownloadAnimeCard
          key={group.animeId}
          animeId={group.animeId}
          animeTitle={group.animeTitle}
          animeImageUrl={group.animeImageUrl}
          downloads={group.downloads}
          onCancel={handleCancel}
          onClearCompleted={clearCompletedForAnime}
        />
      ))}
    </div>
  )
}
