import { useState, useMemo } from "react"
import { Download } from "lucide-react"
import type { Anime, Season } from "@/api/types"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { EpisodeItem } from "./episode-item"

interface EpisodeListProps {
  anime: Anime
  onDownload: (animeId: string, episodeIds: string[]) => void
  isDownloading: boolean
}

function buildTabLabel(season: Season): string {
  if (season.type === "movie") return "Filmes"
  if (season.type === "ova") return "OVAs"
  return season.label
}

export function EpisodeList({ anime, onDownload, isDownloading }: EpisodeListProps) {
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const tabs = useMemo(() => anime.seasons.map((s) => ({
    key: `${s.type}-${s.number}`,
    label: buildTabLabel(s),
    season: s,
  })), [anime.seasons])

  const defaultTab = tabs[0]?.key ?? ""

  const toggleEpisode = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const toggleAllInSeason = (season: Season) => {
    const episodeIds = season.episodes.map((e) => e.id)
    const allSelected = episodeIds.every((id) => selected.has(id))
    setSelected((prev) => {
      const next = new Set(prev)
      for (const id of episodeIds) {
        if (allSelected) {
          next.delete(id)
        } else {
          next.add(id)
        }
      }
      return next
    })
  }

  const handleDownload = () => {
    if (selected.size === 0) return
    onDownload(anime.id, Array.from(selected))
    setSelected(new Set())
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider">Episódios</h3>
        <Button onClick={handleDownload} disabled={selected.size === 0 || isDownloading} size="sm">
          <Download className="h-4 w-4" />
          Baixar {selected.size > 0 ? `(${selected.size})` : "selecionados"}
        </Button>
      </div>

      <Tabs defaultValue={defaultTab}>
        <TabsList>
          {tabs.map((tab) => (
            <TabsTrigger key={tab.key} value={tab.key}>{tab.label}</TabsTrigger>
          ))}
        </TabsList>

        {tabs.map((tab) => (
          <TabsContent key={tab.key} value={tab.key}>
            <div className="space-y-1">
              <button
                type="button"
                onClick={() => toggleAllInSeason(tab.season)}
                className="mb-2 text-xs text-primary hover:underline"
              >
                {tab.season.episodes.every((e) => selected.has(e.id)) ? "Desmarcar tudo" : "Selecionar tudo"}
              </button>
              <div className="max-h-[400px] space-y-0.5 overflow-y-auto">
                {tab.season.episodes.map((ep) => (
                  <EpisodeItem
                    key={ep.id}
                    episode={ep}
                    checked={selected.has(ep.id)}
                    onToggle={toggleEpisode}
                  />
                ))}
              </div>
            </div>
          </TabsContent>
        ))}
      </Tabs>
    </div>
  )
}
