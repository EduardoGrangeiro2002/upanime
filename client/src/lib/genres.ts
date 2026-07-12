import type { Anime } from "@/api/types"

export function groupByGenre(animes: Anime[]): [string, Anime[]][] {
  const groups = new Map<string, Anime[]>()
  let hasClassified = false

  for (const anime of animes) {
    const genres = anime.genres && anime.genres.length > 0 ? anime.genres : null
    if (!genres) continue
    hasClassified = true
    for (const genre of genres) {
      const list = groups.get(genre) ?? []
      list.push(anime)
      groups.set(genre, list)
    }
  }

  if (!hasClassified) return []

  const unclassified = animes.filter((anime) => !anime.genres || anime.genres.length === 0)
  const rows: [string, Anime[]][] = Array.from(groups.entries()).sort((a, b) =>
    a[0].localeCompare(b[0], "pt-BR"),
  )
  if (unclassified.length > 0) {
    rows.push(["Sem categoria", unclassified])
  }
  return rows
}
