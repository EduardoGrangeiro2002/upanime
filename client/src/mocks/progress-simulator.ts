import { useDownloads } from "@/hooks/use-downloads"
import type { Download } from "@/api/types"

const activeTimers = new Map<string, ReturnType<typeof setInterval>>()

function randomBetween(min: number, max: number): number {
  return Math.floor(Math.random() * (max - min + 1)) + min
}

function formatSpeed(mbps: number): string {
  return `${mbps.toFixed(1)} MB/s`
}

function formatEta(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}m ${secs}s`
}

export function simulateDownload(download: Download): void {
  const { setStatus, updateProgress } = useDownloads.getState()

  const queueDelay = randomBetween(200, 500)
  setTimeout(() => {
    setStatus(download.id, "resolving")

    const resolveDelay = randomBetween(500, 1500)
    setTimeout(() => {
      setStatus(download.id, "downloading")

      let progress = 0
      const interval = setInterval(() => {
        const increment = randomBetween(2, 8)
        progress = Math.min(progress + increment, 100)
        const speed = randomBetween(15, 80) / 10
        const remaining = Math.max(0, Math.ceil((100 - progress) / (speed * 2)))

        updateProgress(download.id, progress, formatSpeed(speed), formatEta(remaining))

        if (progress >= 100) {
          clearInterval(interval)
          activeTimers.delete(download.id)
          setStatus(download.id, "completed")
        }
      }, randomBetween(300, 600))

      activeTimers.set(download.id, interval)
    }, resolveDelay)
  }, queueDelay)
}

export function simulateDownloads(downloads: Download[]): void {
  for (const dl of downloads) {
    simulateDownload(dl)
  }
}

export function cancelSimulation(downloadId: string): void {
  const timer = activeTimers.get(downloadId)
  if (!timer) return
  clearInterval(timer)
  activeTimers.delete(downloadId)
}

export function cancelAllSimulations(): void {
  for (const timer of activeTimers.values()) {
    clearInterval(timer)
  }
  activeTimers.clear()
}
