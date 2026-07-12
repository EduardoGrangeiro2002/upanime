import { describe, it, expect, beforeEach } from "vitest"
import { useDownloads } from "../../src/hooks/use-downloads"
import type { Download } from "../../src/api/types"

function createMockDownload(overrides: Partial<Download> = {}): Download {
  return {
    id: "dl-1",
    animeId: "test-anime",
    animeTitle: "Test Anime",
    animeImageUrl: "https://placehold.co/300x450",
    episodeId: "ep-1",
    episodeTitle: "Episódio 1",
    seasonNumber: 1,
    episodeNumber: "1",
    status: "queued",
    progress: 0,
    speed: "",
    eta: "",
    ...overrides,
  }
}

describe("useDownloads store", () => {
  beforeEach(() => {
    useDownloads.setState({ downloads: {} })
  })

  it("adds downloads to the store", () => {
    const dl = createMockDownload()
    useDownloads.getState().addDownloads([dl])
    expect(useDownloads.getState().downloads["dl-1"]).toEqual(dl)
  })

  it("updates progress for an existing download", () => {
    useDownloads.getState().addDownloads([createMockDownload()])
    useDownloads.getState().updateProgress("dl-1", 50, "3.2 MB/s", "1m 20s")
    const dl = useDownloads.getState().downloads["dl-1"]
    expect(dl.progress).toBe(50)
    expect(dl.speed).toBe("3.2 MB/s")
    expect(dl.eta).toBe("1m 20s")
    expect(dl.status).toBe("downloading")
  })

  it("does not change state when updating non-existent download", () => {
    const before = useDownloads.getState().downloads
    useDownloads.getState().updateProgress("nonexistent", 50, "", "")
    expect(useDownloads.getState().downloads).toBe(before)
  })

  it("sets status to completed and progress to 100", () => {
    useDownloads.getState().addDownloads([createMockDownload({ progress: 75 })])
    useDownloads.getState().setStatus("dl-1", "completed")
    const dl = useDownloads.getState().downloads["dl-1"]
    expect(dl.status).toBe("completed")
    expect(dl.progress).toBe(100)
  })

  it("sets status to failed with error message", () => {
    useDownloads.getState().addDownloads([createMockDownload()])
    useDownloads.getState().setStatus("dl-1", "failed", "Network error")
    const dl = useDownloads.getState().downloads["dl-1"]
    expect(dl.status).toBe("failed")
    expect(dl.error).toBe("Network error")
  })

  it("removes a download", () => {
    useDownloads.getState().addDownloads([createMockDownload()])
    useDownloads.getState().removeDownload("dl-1")
    expect(useDownloads.getState().downloads["dl-1"]).toBeUndefined()
  })

  it("clears only completed downloads", () => {
    useDownloads.getState().addDownloads([
      createMockDownload({ id: "dl-1", status: "completed" }),
      createMockDownload({ id: "dl-2", status: "downloading" }),
      createMockDownload({ id: "dl-3", status: "completed" }),
    ])
    useDownloads.getState().clearCompleted()
    const remaining = Object.keys(useDownloads.getState().downloads)
    expect(remaining).toEqual(["dl-2"])
  })
})
