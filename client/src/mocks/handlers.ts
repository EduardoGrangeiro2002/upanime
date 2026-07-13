import { http, HttpResponse, delay } from "msw"
import { createCatalogAnimes, findAnimeByUrl, mockAnimes } from "./data/animes"
import { createDownload } from "./data/downloads"
import type { DownloadRequest, TargetHeight, UpscaleJob } from "@/api/types"

let catalogAnimes = createCatalogAnimes()
let upscaleJobs: UpscaleJob[] = []
let mockUsers = [
  { email: "admin@upanime.dev", isAdmin: true, pending: false },
]

export function resetMockState() {
  catalogAnimes = createCatalogAnimes()
  upscaleJobs = []
  mockUsers = [{ email: "admin@upanime.dev", isAdmin: true, pending: false }]
}

function findCatalogEpisode(id: string) {
  for (const anime of catalogAnimes) {
    for (const season of anime.seasons) {
      for (const episode of season.episodes) {
        if (episode.id === id) {
          return { anime, episode }
        }
      }
    }
  }
  return null
}

export const handlers = [
  http.post("/api/auth/login", async ({ request }) => {
    const { password } = (await request.json()) as { email: string; password: string }
    if (password === "senha-temporaria") {
      return HttpResponse.json({ step: "change_password" })
    }
    if (password === "senha-valida") {
      return HttpResponse.json({ step: "mfa" })
    }
    if (password === "senha-confiavel") {
      return HttpResponse.json({ step: "ok" })
    }
    return HttpResponse.json({ error: "email ou senha inválidos" }, { status: 401 })
  }),

  http.post("/api/auth/change-password", async ({ request }) => {
    const { newPassword } = (await request.json()) as { newPassword: string }
    if (newPassword.length < 8) {
      return HttpResponse.json({ error: "a senha deve ter pelo menos 8 caracteres" }, { status: 400 })
    }
    return HttpResponse.json({ step: "mfa" })
  }),

  http.post("/api/auth/mfa", async ({ request }) => {
    const { code } = (await request.json()) as { code: string }
    if (code === "123456") {
      return HttpResponse.json({ step: "ok" })
    }
    return HttpResponse.json({ error: "código inválido ou expirado" }, { status: 401 })
  }),

  http.post("/api/auth/forgot", async () => {
    return HttpResponse.json({ status: "ok" })
  }),

  http.post("/api/auth/reset", async ({ request }) => {
    const { code } = (await request.json()) as { code: string }
    if (code === "123456") {
      return HttpResponse.json({ status: "ok" })
    }
    return HttpResponse.json({ error: "código inválido ou expirado" }, { status: 401 })
  }),

  http.post("/api/auth/logout", async () => {
    return new HttpResponse(null, { status: 204 })
  }),

  http.get("/api/auth/me", async () => {
    return HttpResponse.json({ email: "admin@upanime.dev", isAdmin: true })
  }),

  http.get("/api/users", async () => {
    return HttpResponse.json(mockUsers)
  }),

  http.post("/api/invites", async ({ request }) => {
    const { email } = (await request.json()) as { email: string }
    if (mockUsers.some((user) => user.email === email)) {
      return HttpResponse.json({ error: "usuário já existe" }, { status: 409 })
    }
    const created = { email, isAdmin: false, pending: true }
    mockUsers.push(created)
    return HttpResponse.json(created)
  }),

  http.get("/api/anime", async ({ request }) => {
    await delay(800)
    const url = new URL(request.url)
    const animeUrl = url.searchParams.get("url")

    if (!animeUrl) {
      return HttpResponse.json({ error: "URL required" }, { status: 400 })
    }

    const anime = findAnimeByUrl(animeUrl)

    if (!anime) {
      return HttpResponse.json({ error: "Anime not found" }, { status: 404 })
    }

    return HttpResponse.json(anime)
  }),

  http.post("/api/downloads", async ({ request }) => {
    await delay(300)
    const body = (await request.json()) as DownloadRequest

    if (!body.episodes?.length) {
      return HttpResponse.json({ error: "episodes required" }, { status: 400 })
    }

    const existing = mockAnimes.find((a) => a.id === body.animeId)
    const animeId = existing?.id ?? `new-${Date.now()}`
    const animeTitle = existing?.title ?? body.animeTitle ?? ""

    if (!existing && !animeTitle) {
      return HttpResponse.json({ error: "animeId or animeTitle required" }, { status: 400 })
    }

    const downloads = body.episodes.map((ep) =>
      createDownload(animeId, animeTitle, body.animeImageUrl, ep.url, ep.title, body.seasonNumber ?? ep.seasonNumber, ep.number),
    )

    return HttpResponse.json(downloads)
  }),

  http.get("/api/downloads", async () => {
    await delay(100)
    return HttpResponse.json([])
  }),

  http.delete("/api/downloads/:id", async () => {
    await delay(100)
    return new HttpResponse(null, { status: 204 })
  }),

  http.get("/api/catalog", async () => {
    await delay(100)
    return HttpResponse.json(catalogAnimes)
  }),

  http.post("/api/catalog/upload", async ({ request }) => {
    await delay(150)
    const form = await request.formData()
    const animeTitle = String(form.get("animeTitle") ?? "").trim()
    const episodeNumber = String(form.get("episodeNumber") ?? "").trim()
    const seasonNumber = parseInt(String(form.get("seasonNumber") ?? "1"), 10)
    const file = form.get("file")

    if (!animeTitle || !episodeNumber || !file || typeof file === "string") {
      return HttpResponse.json({ error: "invalid upload" }, { status: 400 })
    }

    const slug = animeTitle.toLowerCase().replace(/\s+/g, "-")
    let anime = catalogAnimes.find((a) => a.id === `upload-${slug}`)
    if (!anime) {
      anime = {
        id: `upload-${slug}`,
        title: animeTitle,
        url: `upload://${slug}`,
        imageUrl: "",
        description: "",
        genres: [],
        seasons: [],
      }
      catalogAnimes = [...catalogAnimes, anime]
    }

    let season = anime.seasons.find((s) => s.number === seasonNumber && s.type === "episode")
    if (!season) {
      season = { number: seasonNumber, label: `Temporada ${seasonNumber}`, type: "episode", episodes: [] }
      anime.seasons.push(season)
    }

    let episode = season.episodes.find((ep) => ep.number === episodeNumber)
    const replaced = Boolean(episode)
    if (!episode) {
      episode = {
        id: `upload-${slug}-s${seasonNumber}e${episodeNumber}`,
        title: `Episódio ${episodeNumber}`,
        number: episodeNumber,
        seasonNumber,
        type: "episode",
        url: `upload://${slug}/s${seasonNumber}e${episodeNumber}`,
      }
      season.episodes.push(episode)
    }
    episode.storageKey = `animes/${slug}/s${seasonNumber}e${episodeNumber}.mp4`

    return HttpResponse.json({ animeId: anime.id, episode, replaced })
  }),

  http.get("/api/catalog/episode/:id/stream", async ({ params, request }) => {
    await delay(50)
    const found = findCatalogEpisode(String(params.id))
    if (!found) {
      return HttpResponse.json({ error: "Episode not found" }, { status: 404 })
    }

    const variant = new URL(request.url).searchParams.get("variant") ?? "original"
    const storageKey = variant === "upscaled" ? found.episode.upscaledStorageKey : found.episode.storageKey
    if (!storageKey) {
      return HttpResponse.json({ error: "Stream not available" }, { status: 404 })
    }

    return HttpResponse.json({ url: `https://cdn.example.com/${storageKey}` })
  }),

  http.post("/api/catalog/anime/:id/organize", async ({ params }) => {
    await delay(100)
    const anime = catalogAnimes.find((item) => item.id === String(params.id))
    if (!anime) {
      return new HttpResponse(null, { status: 404 })
    }

    for (const season of anime.seasons) {
      for (const episode of season.episodes) {
        const match = episode.title.match(/Epis[oó]dio\s*(\d+)/i)
        if (match) episode.number = String(parseInt(match[1], 10))
      }
      season.episodes.sort((a, b) => Number(a.number) - Number(b.number))
    }
    return HttpResponse.json(anime)
  }),

  http.delete("/api/catalog/episode/:id", async ({ params }) => {
    await delay(50)
    const found = findCatalogEpisode(String(params.id))
    if (!found) {
      return new HttpResponse(null, { status: 404 })
    }

    found.episode.storageKey = undefined
    found.episode.upscaledStorageKey = undefined
    return new HttpResponse(null, { status: 204 })
  }),

  http.delete("/api/catalog/episode/:id/upscaled", async ({ params }) => {
    await delay(50)
    const found = findCatalogEpisode(String(params.id))
    if (!found) {
      return new HttpResponse(null, { status: 404 })
    }

    found.episode.upscaledStorageKey = undefined
    return new HttpResponse(null, { status: 204 })
  }),

  http.post("/api/upscale", async ({ request }) => {
    await delay(100)
    const body = (await request.json()) as { animeId: string; episodeIds: string[]; targetHeight?: TargetHeight }
    const anime = catalogAnimes.find((item) => item.id === body.animeId)
    if (!anime) {
      return HttpResponse.json({ error: "Anime not found" }, { status: 404 })
    }

    const created = body.episodeIds.flatMap((episodeId) => {
      const found = findCatalogEpisode(episodeId)
      if (!found) {
        return []
      }

      const resultStorageKey = found.episode.storageKey?.replace(".mp4", "_upscaled.mp4") ?? ""
      const job: UpscaleJob = {
        id: `job-${upscaleJobs.length + 1}`,
        animeId: anime.id,
        animeImageUrl: anime.imageUrl,
        animeTitle: anime.title,
        episodeId: found.episode.id,
        episodeTitle: found.episode.title,
        episodeNumber: found.episode.number,
        seasonNumber: found.episode.seasonNumber,
        targetHeight: body.targetHeight ?? 1080,
        type: "upscale",
        status: "queued",
        resultStorageKey,
      }
      upscaleJobs = [job, ...upscaleJobs]
      return [job]
    })

    return HttpResponse.json(created)
  }),

  http.get("/api/upscale", async () => {
    await delay(50)
    return HttpResponse.json(upscaleJobs)
  }),

  http.delete("/api/upscale/:id", async ({ params }) => {
    await delay(50)
    upscaleJobs = upscaleJobs.filter((job) => job.id !== String(params.id))
    return new HttpResponse(null, { status: 204 })
  }),
]
