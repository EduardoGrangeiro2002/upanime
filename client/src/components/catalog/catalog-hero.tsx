import type { Anime } from "@/api/types"
import { Play, Info } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"

interface CatalogHeroProps {
  anime: Anime
  onWatch: (anime: Anime) => void
  onSelect: (anime: Anime) => void
}

function countDownloaded(anime: Anime): number {
  let count = 0
  for (const s of anime.seasons) {
    for (const ep of s.episodes) {
      if (ep.storageKey) count++
    }
  }
  return count
}

export function CatalogHero({ anime, onWatch, onSelect }: CatalogHeroProps) {
  const coverSrc = anime.coverUrl || anime.imageUrl
  const downloaded = countDownloaded(anime)

  return (
    <div className="relative h-[300px] md:h-[520px] w-full overflow-hidden">
      {coverSrc && (
        <img
          key={anime.id}
          src={coverSrc}
          alt={anime.title}
          className="absolute inset-0 h-full w-full object-cover object-center opacity-30 scale-105 animate-in fade-in duration-500"
        />
      )}

      <div className="absolute inset-0 bg-gradient-to-t from-background via-background/60 to-transparent" />
      <div className="absolute inset-0 bg-gradient-to-r from-background/80 via-transparent to-transparent" />

      <div className="relative z-10 flex h-full items-end px-4 md:px-8 pb-16 md:pb-24">
        <div key={anime.id} className="max-w-lg space-y-2 md:space-y-4 animate-in fade-in slide-in-from-bottom-2 duration-300">
          <h1 className="font-display text-2xl md:text-5xl font-bold tracking-tighter leading-tight">{anime.title}</h1>
          {anime.genres && anime.genres.length > 0 && (
            <div className="flex flex-wrap gap-1.5">
              {anime.genres.map((genre) => (
                <Badge key={genre} variant="outline" className="text-[11px]">{genre}</Badge>
              ))}
            </div>
          )}
          {anime.description && (
            <p className="text-sm text-muted-foreground line-clamp-2">{anime.description}</p>
          )}

          <div className="flex items-center gap-2 md:gap-3 pt-1">
            <Button
              variant="gradient"
              className="gap-2 hover:neon-primary"
              onClick={() => onWatch(anime)}
            >
              <Play className="h-4 w-4 fill-current" />
              Assistir
            </Button>
            <Button
              variant="glass"
              className="gap-2"
              onClick={() => onSelect(anime)}
            >
              <Info className="h-4 w-4" />
              Mais Info
            </Button>
            <Badge variant="secondary" className="hidden md:inline-flex">
              {downloaded} episódio{downloaded !== 1 ? "s" : ""} disponíve{downloaded !== 1 ? "is" : "l"}
            </Badge>
          </div>
        </div>
      </div>
    </div>
  )
}
