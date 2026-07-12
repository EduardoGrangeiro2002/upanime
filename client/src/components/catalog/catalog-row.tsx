import { useRef } from "react"
import type { Anime } from "@/api/types"
import { ChevronLeft, ChevronRight } from "lucide-react"
import { Button } from "@/components/ui/button"
import { CatalogCard } from "./catalog-card"

interface CatalogRowProps {
  title: string
  animes: Anime[]
  onSelect: (anime: Anime) => void
  onHover: (anime: Anime) => void
}

export function CatalogRow({ title, animes, onSelect, onHover }: CatalogRowProps) {
  const scrollRef = useRef<HTMLDivElement>(null)

  const scroll = (direction: "left" | "right") => {
    if (!scrollRef.current) return
    const amount = scrollRef.current.clientWidth * 0.75
    scrollRef.current.scrollBy({
      left: direction === "left" ? -amount : amount,
      behavior: "smooth",
    })
  }

  return (
    <div className="space-y-3 px-4 md:px-8">
      <h2 className="font-display text-xl font-bold tracking-tight">{title}</h2>

      <div className="group/row relative">
        <Button
          variant="ghost"
          size="icon"
          aria-label="Rolar para a esquerda"
          className="hidden md:flex absolute left-0 top-1/2 -translate-y-1/2 z-20 h-full w-10 rounded-none glass opacity-0 group-hover/row:opacity-100 focus-visible:opacity-100 transition-opacity"
          onClick={() => scroll("left")}
        >
          <ChevronLeft className="h-5 w-5" aria-hidden="true" />
        </Button>

        <div
          ref={scrollRef}
          className="flex gap-4 overflow-x-auto scrollbar-hide scroll-smooth"
        >
          {animes.map((anime) => (
            <CatalogCard
              key={anime.id}
              anime={anime}
              onSelect={onSelect}
              onHover={onHover}
            />
          ))}
        </div>

        <Button
          variant="ghost"
          size="icon"
          aria-label="Rolar para a direita"
          className="hidden md:flex absolute right-0 top-1/2 -translate-y-1/2 z-20 h-full w-10 rounded-none glass opacity-0 group-hover/row:opacity-100 focus-visible:opacity-100 transition-opacity"
          onClick={() => scroll("right")}
        >
          <ChevronRight className="h-5 w-5" aria-hidden="true" />
        </Button>
      </div>
    </div>
  )
}
