import type { Download } from "@/api/types"

let downloadCounter = 0

export function createDownload(
  animeId: string,
  animeTitle: string,
  animeImageUrl: string,
  episodeId: string,
  episodeTitle: string,
  seasonNumber: number,
  episodeNumber: string,
): Download {
  downloadCounter++
  return {
    id: `dl-${downloadCounter}-${Date.now()}`,
    animeId,
    animeTitle,
    animeImageUrl,
    episodeId,
    episodeTitle,
    seasonNumber,
    episodeNumber,
    status: "queued",
    progress: 0,
    speed: "",
    eta: "",
  }
}
