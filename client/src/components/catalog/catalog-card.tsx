import { useRef } from "react"
import type { Anime } from "@/api/types"
import { Play, ImagePlus, Loader2 } from "lucide-react"
import { useUploadCover } from "@/hooks/use-catalog"

interface CatalogCardProps {
  anime: Anime
  onSelect: (anime: Anime) => void
  onHover: (anime: Anime) => void
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

export function CatalogCard({ anime, onSelect, onHover }: CatalogCardProps) {
  const coverSrc = anime.coverUrl || anime.imageUrl
  const downloaded = countDownloaded(anime)
  const fileRef = useRef<HTMLInputElement>(null)
  const uploadCover = useUploadCover()

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    uploadCover.mutate({ animeId: anime.id, file })
    e.target.value = ""
  }

  return (
    <div className="group relative shrink-0 w-[140px] md:w-[200px]">
      <input
        ref={fileRef}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        className="hidden"
        onChange={handleFileChange}
      />

      <button
        onClick={() => onSelect(anime)}
        onMouseEnter={() => onHover(anime)}
        aria-label={`Abrir ${anime.title}`}
        className="block w-full cursor-pointer rounded-xl overflow-hidden transition-all duration-200 hover:scale-[1.03] hover:z-10 hover:shadow-[0_0_20px_rgba(255,92,146,0.15)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <div className="relative aspect-[2/3] bg-muted">
          {coverSrc ? (
            <img
              src={coverSrc}
              alt=""
              className="h-full w-full object-cover"
            />
          ) : (
            <div className="flex h-full flex-col items-center justify-center gap-2 bg-surface-high">
              <span className="font-display text-2xl font-bold text-muted-foreground">{anime.title.charAt(0)}</span>
            </div>
          )}

          <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-black/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-200" />

          <div className="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity duration-200">
            <div className="flex h-11 w-11 items-center justify-center rounded-full bg-gradient-to-r from-primary to-primary-dim shadow-lg">
              <Play className="h-5 w-5 fill-primary-foreground text-primary-foreground ml-0.5" />
            </div>
          </div>
        </div>
      </button>

      {!coverSrc && (
        <button
          type="button"
          onClick={() => fileRef.current?.click()}
          disabled={uploadCover.isPending}
          aria-label={`Enviar capa para ${anime.title}`}
          data-tooltip="Enviar capa"
          className="absolute top-2 right-2 z-10 flex h-8 w-8 items-center justify-center rounded-lg glass text-muted-foreground hover:text-primary transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          {uploadCover.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          ) : (
            <ImagePlus className="h-4 w-4" aria-hidden="true" />
          )}
        </button>
      )}

      <div className="pt-2 px-0.5">
        <p className="text-sm font-medium truncate leading-tight" title={anime.title}>{anime.title}</p>
        <p className="text-[11px] text-muted-foreground mt-0.5">
          {downloaded} ep{downloaded !== 1 ? "s" : ""} · {anime.seasons.length} temp{anime.seasons.length !== 1 ? "s" : "."}
        </p>
      </div>
    </div>
  )
}
