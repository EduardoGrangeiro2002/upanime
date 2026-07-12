import { Checkbox } from "@/components/ui/checkbox"
import type { Episode } from "@/api/types"

interface EpisodeItemProps {
  episode: Episode
  checked: boolean
  onToggle: (id: string) => void
}

export function EpisodeItem({ episode, checked, onToggle }: EpisodeItemProps) {
  return (
    <label className="flex cursor-pointer items-center gap-3 rounded-md px-3 py-2 transition-colors hover:bg-muted">
      <Checkbox checked={checked} onChange={() => onToggle(episode.id)} />
      <span className="min-w-[40px] text-xs tabular-nums text-muted-foreground">
        {episode.type === "episode" ? `${episode.seasonNumber}x${episode.number}` : ""}
      </span>
      <span className="text-sm">{episode.title}</span>
    </label>
  )
}
