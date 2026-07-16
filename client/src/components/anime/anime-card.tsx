import type { Anime } from "@/api/types"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"

interface AnimeCardProps {
  anime: Anime | undefined
  isLoading: boolean
}

export function AnimeCardSkeleton() {
  return (
    <Card>
      <CardContent className="flex gap-4 p-4">
        <Skeleton className="h-[180px] w-[120px] shrink-0 rounded-lg" />
        <div className="flex-1 space-y-3">
          <Skeleton className="h-6 w-3/4" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-2/3" />
        </div>
      </CardContent>
    </Card>
  )
}

export function AnimeCard({ anime, isLoading }: AnimeCardProps) {
  if (isLoading) return <AnimeCardSkeleton />
  if (!anime) return null

  return (
    <Card>
      <CardContent className="flex gap-4 p-4">
        <img
          src={anime.imageUrl}
          alt={anime.title}
          className="h-[180px] w-[120px] shrink-0 rounded-lg object-cover"
        />
        <div className="flex-1 min-w-0 space-y-2">
          <h2 className="text-xl font-bold">{anime.title}</h2>
          <p className="text-sm text-muted-foreground leading-relaxed">{anime.description}</p>
          <div className="flex flex-wrap gap-x-2 gap-y-1 pt-1">
            {anime.seasons.map((s) => (
              <span key={`${s.type}-${s.number}`} className="text-xs text-muted-foreground">
                {s.label}: {s.episodes.length} eps
              </span>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
