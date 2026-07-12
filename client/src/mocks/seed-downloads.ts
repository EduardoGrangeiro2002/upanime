import type { Download, DownloadStatus } from "@/api/types"
import { useDownloads } from "@/hooks/use-downloads"
import { simulateDownloads } from "./progress-simulator"

function makeDl(
  id: string,
  animeId: string,
  animeTitle: string,
  animeImageUrl: string,
  epTitle: string,
  seasonNumber: number,
  epNumber: string,
  status: DownloadStatus,
  progress: number,
  speed = "",
  eta = "",
  error?: string,
): Download {
  return {
    id,
    animeId,
    animeTitle,
    animeImageUrl,
    episodeId: `${animeId}-ep-${epNumber}`,
    episodeTitle: epTitle,
    seasonNumber,
    episodeNumber: epNumber,
    status,
    progress,
    speed,
    eta,
    error,
  }
}

const SNK_IMG = "https://placehold.co/300x450/1a1a2e/f472b6?text=SnK"
const OPM_IMG = "https://placehold.co/300x450/1a1a2e/f472b6?text=OPM"
const JJK_IMG = "https://placehold.co/300x450/1e1e30/a78bfa?text=JJK"

export function seedDownloads() {
  const staticDownloads: Download[] = [
    makeDl("seed-snk-1", "shingeki-no-kyojin", "Shingeki no Kyojin", SNK_IMG, "Episódio 1 - Ao Restante da Humanidade", 1, "1", "completed", 100),
    makeDl("seed-snk-2", "shingeki-no-kyojin", "Shingeki no Kyojin", SNK_IMG, "Episódio 2 - Naquele Dia", 1, "2", "completed", 100),
    makeDl("seed-snk-3", "shingeki-no-kyojin", "Shingeki no Kyojin", SNK_IMG, "Episódio 3 - Uma Fraca Luz na Escuridão", 1, "3", "downloading", 67, "4.2 MB/s", "32s"),
    makeDl("seed-snk-4", "shingeki-no-kyojin", "Shingeki no Kyojin", SNK_IMG, "Episódio 4 - A Noite da Cerimônia", 1, "4", "downloading", 23, "3.8 MB/s", "1m 45s"),
    makeDl("seed-snk-5", "shingeki-no-kyojin", "Shingeki no Kyojin", SNK_IMG, "Episódio 5 - Primeira Batalha", 1, "5", "queued", 0),
    makeDl("seed-snk-6", "shingeki-no-kyojin", "Shingeki no Kyojin", SNK_IMG, "Episódio 6 - O Mundo que Ela Viu", 1, "6", "queued", 0),

    makeDl("seed-opm-1", "one-punch-man", "One Punch Man", OPM_IMG, "Episódio 1 - O Homem Mais Forte", 1, "1", "completed", 100),
    makeDl("seed-opm-2", "one-punch-man", "One Punch Man", OPM_IMG, "Episódio 2 - O Ciborgue Solitário", 1, "2", "downloading", 89, "5.1 MB/s", "8s"),
    makeDl("seed-opm-3", "one-punch-man", "One Punch Man", OPM_IMG, "Episódio 3 - O Cientista Obsessivo", 1, "3", "downloading", 45, "3.5 MB/s", "52s"),
    makeDl("seed-opm-4", "one-punch-man", "One Punch Man", OPM_IMG, "Episódio 4 - O Ninja Moderno", 1, "4", "resolving", 0),

    makeDl("seed-jjk-1", "jujutsu-kaisen", "Jujutsu Kaisen", JJK_IMG, "Episódio 1 - Ryoumen Sukuna", 1, "1", "completed", 100),
    makeDl("seed-jjk-2", "jujutsu-kaisen", "Jujutsu Kaisen", JJK_IMG, "Episódio 2 - Para Mim", 1, "2", "completed", 100),
    makeDl("seed-jjk-3", "jujutsu-kaisen", "Jujutsu Kaisen", JJK_IMG, "Episódio 3 - Garota de Aço", 1, "3", "failed", 34, "", "", "Erro de conexão"),
    makeDl("seed-jjk-4", "jujutsu-kaisen", "Jujutsu Kaisen", JJK_IMG, "Episódio 4 - Maldição", 1, "4", "queued", 0),
  ]

  useDownloads.getState().addDownloads(staticDownloads)

  const toSimulate = staticDownloads.filter((d) => d.status === "downloading" || d.status === "resolving" || d.status === "queued")
  simulateDownloads(toSimulate)
}
