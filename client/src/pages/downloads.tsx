import { useState } from "react"
import { toast } from "sonner"
import { UrlInput } from "@/components/anime/url-input"
import { UploadForm } from "@/components/anime/upload-form"
import { AnimeCard } from "@/components/anime/anime-card"
import { EpisodeList } from "@/components/anime/episode-list"
import { DownloadQueue } from "@/components/download/download-queue"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { useAnime } from "@/hooks/use-anime"
import { useDownloads } from "@/hooks/use-downloads"
import { startDownloads } from "@/api/endpoints"
import { useDownloadPolling } from "@/hooks/use-download-polling"

export function DownloadsPage() {
  useDownloadPolling()
  const [animeUrl, setAnimeUrl] = useState("")
  const [isDownloading, setIsDownloading] = useState(false)
  const { data: anime, isLoading, isError } = useAnime(animeUrl)
  const addDownloads = useDownloads((s) => s.addDownloads)

  const handleDownload = async (animeId: string, episodeIds: string[]) => {
    if (!anime) return
    setIsDownloading(true)
    try {
      const downloads = await startDownloads({
        animeId,
        animeTitle: anime.title,
        animeImageUrl: anime.imageUrl,
        episodeIds,
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
