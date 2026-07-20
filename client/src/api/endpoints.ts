import { apiFetch } from "./client"
import type { Anime, AuthStep, DatasetSample, DatasetStats, DatasetVerdict, Download, DownloadRequest, EpisodeProgress, EpisodeStreamVariant, Me, UploadEpisodeParams, UploadEpisodeResponse, UpscaleJob, UpscaleRequest, UserSummary, WatchProgressItem } from "./types"

export function authLogin(email: string, password: string): Promise<{ step: AuthStep }> {
  return apiFetch<{ step: AuthStep }>("/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  })
}

export function authChangePassword(email: string, currentPassword: string, newPassword: string): Promise<{ step: AuthStep }> {
  return apiFetch<{ step: AuthStep }>("/auth/change-password", {
    method: "POST",
    body: JSON.stringify({ email, currentPassword, newPassword }),
  })
}

export function authVerifyMfa(email: string, code: string): Promise<{ step: AuthStep }> {
  return apiFetch<{ step: AuthStep }>("/auth/mfa", {
    method: "POST",
    body: JSON.stringify({ email, code }),
  })
}

export function authForgot(email: string): Promise<{ status: string }> {
  return apiFetch<{ status: string }>("/auth/forgot", {
    method: "POST",
    body: JSON.stringify({ email }),
  })
}

export function authReset(email: string, code: string, newPassword: string): Promise<{ status: string }> {
  return apiFetch<{ status: string }>("/auth/reset", {
    method: "POST",
    body: JSON.stringify({ email, code, newPassword }),
  })
}

export function authLogout(): Promise<void> {
  return apiFetch<void>("/auth/logout", { method: "POST" })
}

export function fetchMe(): Promise<Me> {
  return apiFetch<Me>("/auth/me")
}

export function fetchUsers(): Promise<UserSummary[]> {
  return apiFetch<UserSummary[]>("/users")
}

export function sendInvite(email: string): Promise<UserSummary> {
  return apiFetch<UserSummary>("/invites", {
    method: "POST",
    body: JSON.stringify({ email }),
  })
}

export function fetchAnime(url: string): Promise<Anime> {
  return apiFetch<Anime>(`/anime?url=${encodeURIComponent(url)}`)
}

export function startDownloads(request: DownloadRequest): Promise<Download[]> {
  return apiFetch<Download[]>("/downloads", {
    method: "POST",
    body: JSON.stringify(request),
  })
}

export function cancelDownload(id: string): Promise<void> {
  return apiFetch<void>(`/downloads/${id}`, { method: "DELETE" })
}

export function fetchDownloads(): Promise<Download[]> {
  return apiFetch<Download[]>("/downloads")
}

export function fetchCatalog(): Promise<Anime[]> {
  return apiFetch<Anime[]>("/catalog")
}

export function deleteAnime(id: string): Promise<void> {
  return apiFetch<void>(`/catalog/anime/${id}`, { method: "DELETE" })
}

export function organizeAnime(id: string): Promise<{ status: string }> {
  return apiFetch<{ status: string }>(`/catalog/anime/${id}/organize`, { method: "POST" })
}

export function deleteEpisode(id: string): Promise<void> {
  return apiFetch<void>(`/catalog/episode/${id}`, { method: "DELETE" })
}

export function streamFileURL(id: string, variant: string = "original"): string {
  return `/api/catalog/episode/${id}/stream/file?variant=${variant}`
}

export function fetchEpisodeStreamURL(id: string, variant: EpisodeStreamVariant = "original"): Promise<{ url: string }> {
  return Promise.resolve({ url: streamFileURL(id, variant) })
}

export function episodeThumbnailURL(id: string): string {
  return `/api/catalog/episode/${id}/thumbnail`
}

export function fetchWatchProgressList(): Promise<WatchProgressItem[]> {
  return apiFetch<WatchProgressItem[]>("/progress")
}

export function fetchEpisodeProgress(id: string): Promise<EpisodeProgress> {
  return apiFetch<EpisodeProgress>(`/progress/episode/${id}`)
}

export function saveEpisodeProgress(id: string, position: number, duration: number): Promise<void> {
  return apiFetch<void>(`/progress/episode/${id}`, {
    method: "PUT",
    body: JSON.stringify({ position, duration }),
    keepalive: true,
  })
}

export function startUpscale(req: UpscaleRequest): Promise<UpscaleJob[]> {
  return apiFetch<UpscaleJob[]>("/upscale", {
    method: "POST",
    body: JSON.stringify(req),
  })
}

export function fetchUpscaleJobs(): Promise<UpscaleJob[]> {
  return apiFetch<UpscaleJob[]>("/upscale")
}

export function deleteUpscaleJob(id: string): Promise<void> {
  return apiFetch<void>(`/upscale/${id}`, { method: "DELETE" })
}

export function deleteUpscaledEpisode(id: string): Promise<void> {
  return apiFetch<void>(`/catalog/episode/${id}/upscaled`, { method: "DELETE" })
}

export function fetchDatasetQueue(limit = 50): Promise<DatasetSample[]> {
  return apiFetch<DatasetSample[]>(`/dataset/samples/queue?limit=${limit}`)
}

export function submitDatasetVerdict(id: string, verdict: DatasetVerdict): Promise<void> {
  return apiFetch<void>(`/dataset/samples/${id}/verdict`, {
    method: "POST",
    body: JSON.stringify({ verdict }),
  })
}

export function fetchDatasetStats(): Promise<DatasetStats> {
  return apiFetch<DatasetStats>("/dataset/stats")
}

export function uploadEpisode(
  params: UploadEpisodeParams,
  onProgress?: (pct: number) => void,
): Promise<UploadEpisodeResponse> {
  return new Promise((resolve, reject) => {
    const form = new FormData()
    form.append("animeTitle", params.animeTitle)
    form.append("seasonNumber", String(params.seasonNumber))
    form.append("episodeNumber", params.episodeNumber)
    form.append("file", params.file)

    const xhr = new XMLHttpRequest()
    xhr.open("POST", "/api/catalog/upload")
    xhr.upload.onprogress = (e) => {
      if (!e.lengthComputable || !onProgress) return
      onProgress(Math.round((e.loaded / e.total) * 100))
    }
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(JSON.parse(xhr.responseText))
        return
      }
      reject(new Error(xhr.responseText || `upload falhou (${xhr.status})`))
    }
    xhr.onerror = () => reject(new Error("falha de rede no upload"))
    xhr.send(form)
  })
}

export async function uploadAnimeCover(animeId: string, file: File): Promise<{ coverUrl: string }> {
  const form = new FormData()
  form.append("cover", file)
  const res = await fetch(`/api/catalog/anime/${animeId}/cover`, {
    method: "POST",
    body: form,
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}
