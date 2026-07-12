import { create } from "zustand"
import type { Download, DownloadStatus } from "@/api/types"

interface DownloadsState {
  downloads: Record<string, Download>
  addDownloads: (downloads: Download[]) => void
  updateProgress: (id: string, progress: number, speed: string, eta: string) => void
  setStatus: (id: string, status: DownloadStatus, error?: string) => void
  removeDownload: (id: string) => void
  clearCompleted: () => void
  clearCompletedForAnime: (animeId: string) => void
  syncFromServer: (serverDownloads: Download[]) => void
}

export const useDownloads = create<DownloadsState>((set) => ({
  downloads: {},

  addDownloads: (downloads) =>
    set((state) => {
      const updated = { ...state.downloads }
      for (const dl of downloads) {
        updated[dl.id] = dl
      }
      return { downloads: updated }
    }),

  updateProgress: (id, progress, speed, eta) =>
    set((state) => {
      const dl = state.downloads[id]
      if (!dl) return state
      return {
        downloads: {
          ...state.downloads,
          [id]: { ...dl, progress, speed, eta, status: "downloading" },
        },
      }
    }),

  setStatus: (id, status, error) =>
    set((state) => {
      const dl = state.downloads[id]
      if (!dl) return state
      return {
        downloads: {
          ...state.downloads,
          [id]: {
            ...dl,
            status,
            error,
            progress: status === "completed" ? 100 : dl.progress,
          },
        },
      }
    }),

  removeDownload: (id) =>
    set((state) => {
      const rest = { ...state.downloads }
      delete rest[id]
      return { downloads: rest }
    }),

  clearCompleted: () =>
    set((state) => {
      const filtered: Record<string, Download> = {}
      for (const [id, dl] of Object.entries(state.downloads)) {
        if (dl.status !== "completed") {
          filtered[id] = dl
        }
      }
      return { downloads: filtered }
    }),

  clearCompletedForAnime: (animeId) =>
    set((state) => {
      const filtered: Record<string, Download> = {}
      for (const [id, dl] of Object.entries(state.downloads)) {
        if (dl.animeId === animeId && dl.status === "completed") continue
        filtered[id] = dl
      }
      return { downloads: filtered }
    }),

  syncFromServer: (serverDownloads) =>
    set((state) => {
      const updated = { ...state.downloads }
      const serverIds = new Set(serverDownloads.map((dl) => dl.id))

      for (const dl of serverDownloads) {
        updated[dl.id] = { ...updated[dl.id], ...dl }
      }

      for (const [id, dl] of Object.entries(updated)) {
        if (serverIds.has(id)) continue
        if (dl.status === "completed" || dl.status === "failed") continue
        updated[id] = { ...dl, status: "completed", progress: 100 }
      }

      return { downloads: updated }
    }),
}))
