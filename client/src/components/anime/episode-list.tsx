import { useState, useMemo } from "react"
import { Download } from "lucide-react"
import type { Anime, Episode, Season } from "@/api/types"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { EpisodeItem } from "./episode-item"

interface EpisodeListProps {
  anime: Anime
  onDownload: (episodes: Episode[]) => void
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

  const toggleEpisode = (url: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(url)) {
        next.delete(url)
      } else {
        next.add(url)
      }
      return next
    })
  }

  const toggleAllInSeason = (season: Season) => {
    const episodeUrls = season.episodes.map((e) => e.url)
    const allSelected = episodeUrls.every((url) => selected.has(url))
    setSelected((prev) => {
      const next = new Set(prev)
      for (const url of episodeUrls) {
        if (allSelected) {
          next.delete(url)
        } else {
          next.add(url)
        }
      }
      return next
    })
  }

  const handleDownload = () => {
    if (selected.size === 0) return
    const episodes = anime.seasons.flatMap((s) => s.episodes).filter((e) => selected.has(e.url))
    onDownload(episodes)
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
                {tab.season.episodes.every((e) => selected.has(e.url)) ? "Desmarcar tudo" : "Selecionar tudo"}
              </button>
              <div className="max-h-[400px] space-y-0.5 overflow-y-auto">
                {tab.season.episodes.map((ep) => (
                  <EpisodeItem
                    key={ep.url}
                    episode={ep}
                    checked={selected.has(ep.url)}
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
