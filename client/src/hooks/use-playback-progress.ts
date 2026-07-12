import { useCallback, useRef } from "react"

const STORAGE_KEY = "upanime:playback-progress"
const SAVE_INTERVAL_MS = 5000
const COMPLETED_THRESHOLD = 0.95

interface ProgressEntry {
  t: number
  d: number
}

type StoredProgress = number | ProgressEntry

interface ProgressMap {
  [episodeId: string]: StoredProgress
}

function loadAll(): ProgressMap {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEY) || "{}")
  } catch {
    return {}
  }
}

function persistAll(map: ProgressMap) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(map))
}

export function normalizeEntry(entry: StoredProgress | undefined): ProgressEntry {
  if (entry === undefined) return { t: 0, d: 0 }
  if (typeof entry === "number") return { t: entry, d: 0 }
  return entry
}

export function getProgress(episodeId: string): number {
  return normalizeEntry(loadAll()[episodeId]).t
}

export function getProgressPct(episodeId: string): number {
  const { t, d } = normalizeEntry(loadAll()[episodeId])
  if (t <= 0) return 0
  if (d <= 0) return 0
  return Math.min(100, Math.round((t / d) * 100))
}

export function clearProgress(episodeId: string) {
  const map = loadAll()
  delete map[episodeId]
  persistAll(map)
}

export function usePlaybackProgress(episodeId: string | null) {
  const lastSaveRef = useRef(0)

  const savedTime = episodeId ? getProgress(episodeId) : 0

  const handleTimeUpdate = useCallback(
    (time: number) => {
      if (!episodeId) return
      const now = Date.now()
      if (now - lastSaveRef.current < SAVE_INTERVAL_MS) return
      lastSaveRef.current = now

      const player = document.querySelector("media-player") as HTMLElement & { duration?: number } | null
      const duration = player?.duration ?? 0
      if (duration > 0 && time / duration > COMPLETED_THRESHOLD) {
        clearProgress(episodeId)
        return
      }

      const map = loadAll()
      map[episodeId] = { t: Math.floor(time), d: Math.floor(duration) }
      persistAll(map)
    },
    [episodeId],
  )

  return { savedTime, handleTimeUpdate }
}
