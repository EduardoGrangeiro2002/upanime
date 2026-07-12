import { useEffect, useRef } from "react"
import { fetchDownloads } from "@/api/endpoints"
import { useDownloads } from "./use-downloads"

export function useDownloadPolling() {
  const syncFromServer = useDownloads((s) => s.syncFromServer)
  const hasActiveDownloads = useDownloads((s) => {
    return Object.values(s.downloads).some(
      (d) => d.status === "queued" || d.status === "resolving" || d.status === "downloading"
    )
  })

  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    fetchDownloads().then(syncFromServer).catch(() => undefined)
  }, [syncFromServer])

  useEffect(() => {
    if (!hasActiveDownloads) {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
      return
    }

    const poll = async () => {
      try {
        const downloads = await fetchDownloads()
        syncFromServer(downloads)
      } catch {
        return
      }
    }

    poll()
    intervalRef.current = setInterval(poll, 2000)

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
    }
  }, [hasActiveDownloads, syncFromServer])
}
