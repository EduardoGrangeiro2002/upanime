import { useEffect, useRef, useState } from "react"
import { Library, RefreshCw } from "lucide-react"
import { useCatalog, useDeleteAnime, useDeleteEpisode, useDeleteUpscaledEpisode } from "@/hooks/use-catalog"
import { useRoute } from "@/hooks/use-route"
import { CatalogHero } from "@/components/catalog/catalog-hero"
import { CatalogRow } from "@/components/catalog/catalog-row"
import { CatalogDetail } from "@/components/catalog/catalog-detail"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import type { Anime } from "@/api/types"
import { groupByGenre } from "@/lib/genres"

export function CatalogPage() {
  const { data: animes, isLoading, error, refetch, isRefetching } = useCatalog()
  const { param: selectedAnimeId, navigate } = useRoute()
  const deleteAnimeMutation = useDeleteAnime()
  const deleteEpisodeMutation = useDeleteEpisode()
  const deleteUpscaledMutation = useDeleteUpscaledEpisode()
  const [featuredAnimeId, setFeaturedAnimeId] = useState<string | null>(null)
  const [playOnOpen, setPlayOnOpen] = useState(false)
  const hoverTimer = useRef<number | null>(null)

  useEffect(() => {
    return () => {
      if (hoverTimer.current) window.clearTimeout(hoverTimer.current)
    }
  }, [])

  const openAnime = (anime: Anime, play: boolean) => {
    setPlayOnOpen(play)
    navigate("catalog", anime.id)
  }

  const closeDetail = () => {
    setPlayOnOpen(false)
    navigate("catalog")
  }

  const handleHover = (anime: Anime) => {
    if (hoverTimer.current) window.clearTimeout(hoverTimer.current)
    hoverTimer.current = window.setTimeout(() => setFeaturedAnimeId(anime.id), 250)
  }

  if (isLoading) {
    return (
      <div className="space-y-8 pb-8">
        <Skeleton className="h-[300px] md:h-[520px] w-full rounded-none" />
        <div className="px-4 md:px-8 space-y-6">
          <Skeleton className="h-5 w-40" />
          <div className="flex gap-4">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-[182px] w-[140px] md:h-[260px] md:w-[200px] shrink-0 rounded-xl" />
            ))}
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-4 py-20 text-muted-foreground">
        <div className="text-center">
          <p className="text-sm font-medium text-foreground">Não foi possível carregar o catálogo</p>
          <p className="text-xs mt-1">Verifique se o servidor está no ar e tente de novo.</p>
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()} disabled={isRefetching}>
          <RefreshCw className="h-3.5 w-3.5" />
          Tentar novamente
        </Button>
      </div>
    )
  }

  if (!animes || animes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
        <Library className="h-12 w-12 mb-3 opacity-40" />
        <p className="font-display text-sm font-medium">Catálogo vazio</p>
        <p className="text-xs mt-1">Baixe episódios para vê-los aqui.</p>
      </div>
    )
  }

  const selectedAnime = animes.find((anime) => anime.id === selectedAnimeId) ?? null
  const featuredAnime = animes.find((anime) => anime.id === featuredAnimeId) ?? null
  const hero = featuredAnime ?? animes[0]
  const genreRows = groupByGenre(animes)

  return (
    <div className="pb-8">
      <CatalogHero
        anime={hero}
        onWatch={(anime) => openAnime(anime, true)}
        onSelect={(anime) => openAnime(anime, false)}
      />

      <div className="-mt-4 md:-mt-16 relative z-10 space-y-8">
        <CatalogRow
          title="Meu Catálogo"
          animes={animes}
          onSelect={(anime) => openAnime(anime, false)}
          onHover={handleHover}
        />

        {genreRows.map(([genre, genreAnimes]) => (
          <CatalogRow
            key={genre}
            title={genre}
            animes={genreAnimes}
            onSelect={(anime) => openAnime(anime, false)}
            onHover={handleHover}
          />
        ))}
      </div>

      {selectedAnime && (
        <CatalogDetail
          anime={selectedAnime}
          autoPlayOnOpen={playOnOpen}
          onClose={closeDetail}
          onDeleteAnime={(id) => {
            deleteAnimeMutation.mutate(id, { onSuccess: closeDetail })
          }}
          onDeleteEpisode={(id) => deleteEpisodeMutation.mutate(id)}
          onDeleteUpscaledEpisode={(id) => deleteUpscaledMutation.mutate(id)}
          isDeletingAnime={deleteAnimeMutation.isPending}
          isDeletingEpisode={deleteEpisodeMutation.isPending}
          isDeletingUpscaledEpisode={deleteUpscaledMutation.isPending}
        />
      )}
    </div>
  )
}
