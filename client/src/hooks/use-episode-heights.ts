import { useEffect, useRef, useState } from "react"

export function useEpisodeHeights(episodeIds: string[]): Map<string, number> {
  const [heights, setHeights] = useState<Map<string, number>>(new Map())
  const pending = useRef(new Set<string>())

  useEffect(() => {
    for (const id of episodeIds) {
      if (heights.has(id)) continue
      if (pending.current.has(id)) continue

      pending.current.add(id)
      const video = document.createElement("video")
      video.preload = "metadata"
      video.onloadedmetadata = () => {
        if (video.videoHeight > 0) {
          setHeights((previous) => new Map(previous).set(id, video.videoHeight))
        }
        video.removeAttribute("src")
      }
      video.onerror = () => {
        pending.current.delete(id)
      }
      video.src = `/api/catalog/episode/${id}/stream/file?variant=original`
    }
  }, [episodeIds, heights])

  return heights
}
