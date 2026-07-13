import { useEffect, useState } from "react"
import { toast } from "sonner"
import { UrlInput } from "@/components/anime/url-input"
import { UploadForm } from "@/components/anime/upload-form"
import { AnimeCard } from "@/components/anime/anime-card"
import { EpisodeList } from "@/components/anime/episode-list"
import { DownloadQueue } from "@/components/download/download-queue"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Input } from "@/components/ui/input"
import { useAnime } from "@/hooks/use-anime"
import { useCatalog } from "@/hooks/use-catalog"
import { useDownloads } from "@/hooks/use-downloads"
import { startDownloads } from "@/api/endpoints"
import { useDownloadPolling } from "@/hooks/use-download-polling"
import type { Episode } from "@/api/types"

export function DownloadsPage() {
  useDownloadPolling()
  const [animeUrl, setAnimeUrl] = useState("")
  const [isDownloading, setIsDownloading] = useState(false)
  const [targetAnimeId, setTargetAnimeId] = useState("")
  const [customTitle, setCustomTitle] = useState<string | null>(null)
  const [targetSeason, setTargetSeason] = useState("")
  const { data: anime, isLoading, isError } = useAnime(animeUrl)
  const { data: catalog } = useCatalog()
  const addDownloads = useDownloads((s) => s.addDownloads)

  useEffect(() => {
    setTargetAnimeId("")
    setCustomTitle(null)
    setTargetSeason("")
  }, [anime?.url])

  const handleDownload = async (episodes: Episode[]) => {
    if (!anime) return

    const newTitle = (customTitle ?? anime.title).trim()
    if (!targetAnimeId && !newTitle) {
      toast.error("Informe o nome do anime ou escolha um do catálogo")
      return
    }

    setIsDownloading(true)
    try {
      const downloads = await startDownloads({
        animeId: targetAnimeId || undefined,
        animeTitle: targetAnimeId ? undefined : newTitle,
        animeImageUrl: anime.imageUrl,
        description: anime.description,
        sourceUrl: anime.url,
        seasonNumber: targetSeason ? Number(targetSeason) : undefined,
        episodes: episodes.map((e) => ({
          title: e.title,
          number: e.number,
          url: e.url,
          seasonNumber: e.seasonNumber,
        })),
      })
      addDownloads(downloads)
      toast.success(
        downloads.length === 1
          ? "1 episódio adicionado à fila"
          : `${downloads.length} episódios adicionados à fila`,
      )
    } catch {
      toast.error("Não foi possível iniciar os downloads", {
        description: "O servidor não respondeu. Tente novamente.",
      })
    } finally {
      setIsDownloading(false)
    }
  }

  return (
    <div className="px-4 md:px-8 py-6">
      <div className="mx-auto max-w-5xl space-y-6">
        <div>
          <h1 className="font-display text-2xl font-bold tracking-tight">Downloads</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Busque um anime por URL ou envie seus próprios arquivos de vídeo.
          </p>
        </div>

        <Tabs defaultValue="url">
          <TabsList>
            <TabsTrigger value="url">Buscar por URL</TabsTrigger>
            <TabsTrigger value="upload">Enviar arquivos</TabsTrigger>
          </TabsList>

          <TabsContent value="url" className="space-y-6">
            <UrlInput onSubmit={setAnimeUrl} isLoading={isLoading} />

            {isError && (
              <div role="alert" className="rounded-xl bg-destructive/10 p-3 text-sm text-destructive">
                Não foi possível encontrar o anime. Verifique se a URL aponta para a página do anime e tente novamente.
              </div>
            )}

            <AnimeCard anime={anime} isLoading={isLoading} />

            {anime && (
              <div className="rounded-xl border border-border p-4 space-y-3">
                <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider">Salvar no catálogo</h3>
                <div className="grid gap-3 sm:grid-cols-3">
                  <div className="space-y-1.5">
                    <label htmlFor="target-anime" className="text-sm font-medium">Anime de destino</label>
                    <select
                      id="target-anime"
                      value={targetAnimeId}
                      onChange={(e) => setTargetAnimeId(e.target.value)}
                      className="flex h-9 w-full rounded-lg bg-input px-3 py-1 text-sm shadow-sm transition-all focus-visible:outline-none focus-visible:border-b-2 focus-visible:border-primary"
                    >
                      <option value="">Criar novo anime</option>
                      {catalog?.map((a) => (
                        <option key={a.id} value={a.id}>{a.title}</option>
                      ))}
                    </select>
                  </div>

                  {!targetAnimeId && (
                    <div className="space-y-1.5">
                      <label htmlFor="new-anime-title" className="text-sm font-medium">Nome do novo anime</label>
                      <Input
                        id="new-anime-title"
                        value={customTitle ?? anime.title}
                        onChange={(e) => setCustomTitle(e.target.value)}
                      />
                    </div>
                  )}

                  <div className="space-y-1.5">
                    <label htmlFor="target-season" className="text-sm font-medium">Temporada de destino</label>
                    <Input
                      id="target-season"
                      type="number"
                      min={1}
                      value={targetSeason}
                      onChange={(e) => setTargetSeason(e.target.value)}
                      placeholder="Manter numeração do site"
                    />
                  </div>
                </div>
              </div>
            )}

            {anime && (
              <EpisodeList
                anime={anime}
                onDownload={handleDownload}
                isDownloading={isDownloading}
              />
            )}
          </TabsContent>

          <TabsContent value="upload">
            <UploadForm />
          </TabsContent>
        </Tabs>

        <DownloadQueue />
      </div>
    </div>
  )
}
