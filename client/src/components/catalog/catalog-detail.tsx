import { useRef, useState } from "react"
import type { Anime, Season } from "@/api/types"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { X, Trash2, ImagePlus, Loader2, Sparkles } from "lucide-react"
import { CatalogEpisodeList } from "./catalog-episode-list"
import { useOrganizeAnime, useUploadCover } from "@/hooks/use-catalog"
import { useDialog } from "@/hooks/use-dialog"

interface CatalogDetailProps {
  anime: Anime
  autoPlayOnOpen?: boolean
  onClose: () => void
  onDeleteAnime: (id: string) => void
  onDeleteEpisode: (id: string) => void
  onDeleteUpscaledEpisode: (id: string) => void
  isDeletingAnime: boolean
  isDeletingEpisode: boolean
  isDeletingUpscaledEpisode: boolean
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

function getSeasonOptions(seasons: Season[]): Season[] {
  return seasons.filter((s) => s.episodes.some((ep) => ep.storageKey))
}

export function CatalogDetail({
  anime,
  autoPlayOnOpen = false,
  onClose,
  onDeleteAnime,
  onDeleteEpisode,
  onDeleteUpscaledEpisode,
  isDeletingAnime,
  isDeletingEpisode,
  isDeletingUpscaledEpisode,
}: CatalogDetailProps) {
  const [confirmDelete, setConfirmDelete] = useState(false)
  const availableSeasons = getSeasonOptions(anime.seasons)
  const [activeSeason, setActiveSeason] = useState(availableSeasons[0]?.number ?? 1)
  const coverSrc = anime.coverUrl || anime.imageUrl
  const downloaded = countDownloaded(anime)
  const fileRef = useRef<HTMLInputElement>(null)
  const uploadCover = useUploadCover()
  const organizeMutation = useOrganizeAnime()
  const dialogRef = useDialog(onClose)

  const currentSeason = availableSeasons.find((s) => s.number === activeSeason) ?? availableSeasons[0]

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    uploadCover.mutate({ animeId: anime.id, file })
    e.target.value = ""
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/70 backdrop-blur-sm pt-0 pb-0 md:pt-8 md:pb-8"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <input
        ref={fileRef}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        className="hidden"
        onChange={handleFileChange}
      />
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="catalog-detail-title"
        className="relative w-full max-w-full md:max-w-3xl rounded-none md:rounded-2xl overflow-hidden bg-surface/95 backdrop-blur-[30px] shadow-2xl animate-in fade-in slide-in-from-bottom-4 duration-300"
      >
        <div className="relative h-[160px] md:h-[280px] overflow-hidden">
          {coverSrc ? (
            <img
              src={coverSrc}
              alt=""
              className="absolute inset-0 h-full w-full object-cover object-center blur-sm scale-110 opacity-50"
            />
          ) : (
            <div className="absolute inset-0 bg-surface-high" />
          )}
          <div className="absolute inset-0 bg-gradient-to-t from-surface via-surface/40 to-transparent" />

          <div className="absolute top-3 right-3 z-20 flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon"
              aria-label="Organizar episódios com IA"
              data-tooltip="Organizar episódios (IA)"
              className="h-9 w-9 rounded-full glass text-white hover:bg-white/10"
              onClick={() => organizeMutation.mutate(anime.id)}
              disabled={organizeMutation.isPending}
            >
              {organizeMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              ) : (
                <Sparkles className="h-4 w-4" aria-hidden="true" />
              )}
            </Button>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Enviar capa personalizada"
              data-tooltip="Enviar capa"
              className="h-9 w-9 rounded-full glass text-white hover:bg-white/10"
              onClick={() => fileRef.current?.click()}
              disabled={uploadCover.isPending}
            >
              {uploadCover.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              ) : (
                <ImagePlus className="h-4 w-4" aria-hidden="true" />
              )}
            </Button>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Fechar detalhes"
              data-tooltip="Fechar (Esc)"
              className="h-9 w-9 rounded-full glass text-white hover:bg-white/10"
              onClick={onClose}
            >
              <X className="h-4 w-4" aria-hidden="true" />
            </Button>
          </div>

          <div className="absolute bottom-0 left-0 right-0 p-4 md:p-6 z-10">
            <h2 id="catalog-detail-title" className="font-display text-xl md:text-3xl font-bold tracking-tight">{anime.title}</h2>
            <div className="flex flex-wrap items-center gap-2 mt-2">
              <Badge variant="secondary" className="text-xs">
                {downloaded} episódio{downloaded !== 1 ? "s" : ""}
              </Badge>
              {(anime.genres ?? []).map((genre) => (
                <Badge key={genre} variant="outline" className="text-[11px]">{genre}</Badge>
              ))}
              {anime.seasons.length > 1 && (
                <span className="text-xs text-muted-foreground">
                  {anime.seasons.length} temporadas
                </span>
              )}
            </div>
            {anime.description && (
              <p className="text-sm text-muted-foreground mt-3 line-clamp-2 max-w-lg">
                {anime.description}
              </p>
            )}
          </div>
        </div>

        <div className="px-4 md:px-6 pt-4 pb-6 space-y-4">
          {availableSeasons.length > 1 && (
            <div className="flex gap-1 overflow-x-auto scrollbar-hide" role="group" aria-label="Temporadas">
              {availableSeasons.map((season) => (
                <button
                  key={season.number}
                  onClick={() => setActiveSeason(season.number)}
                  aria-pressed={activeSeason === season.number}
                  className={`shrink-0 px-3 py-1.5 rounded-full text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
                    activeSeason === season.number
                      ? "bg-primary text-primary-foreground"
                      : "bg-surface-high text-muted-foreground hover:bg-surface-highest"
                  }`}
                >
                  {season.label}
                </button>
              ))}
            </div>
          )}

          {currentSeason && (
            <CatalogEpisodeList
              animeTitle={anime.title}
              season={currentSeason}
              autoPlayOnOpen={autoPlayOnOpen}
              onDeleteEpisode={onDeleteEpisode}
              onDeleteUpscaledEpisode={onDeleteUpscaledEpisode}
              isDeleting={isDeletingEpisode}
              isDeletingUpscaled={isDeletingUpscaledEpisode}
            />
          )}

          <div className="border-t border-white/[0.06] pt-4 flex justify-end">
            {confirmDelete ? (
              <div className="flex gap-2">
                <Button
                  variant="destructive"
                  size="sm"
                  disabled={isDeletingAnime}
                  onClick={() => onDeleteAnime(anime.id)}
                >
                  Deletar anime e episódios
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setConfirmDelete(false)}
                >
                  Cancelar
                </Button>
              </div>
            ) : (
              <Button
                variant="ghost"
                size="sm"
                className="text-muted-foreground hover:text-destructive"
                onClick={() => setConfirmDelete(true)}
              >
                <Trash2 className="h-3.5 w-3.5 mr-1.5" aria-hidden="true" />
                Remover do catálogo
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
