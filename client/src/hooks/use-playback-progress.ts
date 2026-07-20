import { useCallback, useEffect, useRef } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { fetchEpisodeProgress, fetchWatchProgressList, saveEpisodeProgress } from "@/api/endpoints"
import type { WatchProgressItem } from "@/api/types"

const REPORT_INTERVAL_MS = 10_000
const MIN_RESUME_SECONDS = 5
const COMPLETED_THRESHOLD = 0.95

export function resumeTime(position: number, duration: number): number {
  if (position <= MIN_RESUME_SECONDS) return 0
  if (duration > 0 && position / duration >= COMPLETED_THRESHOLD) return 0
  return position
}

export function progressPct(position: number, duration: number): number {
  if (position <= 0) return 0
  if (duration <= 0) return 0
  return Math.min(100, Math.round((position / duration) * 100))
}

export function buildProgressMap(items: WatchProgressItem[] | undefined): Record<string, WatchProgressItem> {
  const map: Record<string, WatchProgressItem> = {}
  for (const item of items ?? []) {
    map[item.episodeId] = item
  }
  return map
}

export function useWatchProgressList() {
  return useQuery({
    queryKey: ["watch-progress"],
    queryFn: fetchWatchProgressList,
    staleTime: 30 * 1000,
    meta: { silentError: true },
  })
}

export function usePlaybackProgress(episodeId: string | null) {
  const queryClient = useQueryClient()
  const lastReportAtRef = useRef(0)
  const latestRef = useRef<{ position: number; duration: number } | null>(null)

  const { data, isFetched } = useQuery({
    queryKey: ["episode-progress", episodeId],
    queryFn: () => fetchEpisodeProgress(episodeId!),
    enabled: episodeId !== null,
    retry: false,
    meta: { silentError: true },
  })

  const report = useCallback(
    (position: number, duration: number) => {
      if (!episodeId) return
      saveEpisodeProgress(episodeId, position, duration)
        .then(() => queryClient.invalidateQueries({ queryKey: ["watch-progress"] }))
        .catch(() => undefined)
    },
    [episodeId, queryClient],
  )

  const handleTimeUpdate = useCallback(
    (position: number, duration: number) => {
      latestRef.current = { position, duration }
      const now = Date.now()
      if (now - lastReportAtRef.current < REPORT_INTERVAL_MS) return
      lastReportAtRef.current = now
      report(position, duration)
    },
    [report],
  )

  const flush = useCallback(() => {
    const latest = latestRef.current
    if (!latest) return
    lastReportAtRef.current = Date.now()
    report(latest.position, latest.duration)
  }, [report])

  useEffect(() => {
    latestRef.current = null
    lastReportAtRef.current = Date.now()
    return flush
  }, [episodeId, flush])

  useEffect(() => {
    window.addEventListener("pagehide", flush)
    return () => window.removeEventListener("pagehide", flush)
  }, [flush])

  const savedTime = data ? resumeTime(data.position, data.duration) : 0

  return { savedTime, ready: episodeId === null || isFetched, handleTimeUpdate, flush }
}
