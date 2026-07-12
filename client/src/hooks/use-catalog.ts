import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { fetchCatalog, deleteAnime, deleteEpisode, deleteUpscaledEpisode, fetchEpisodeStreamURL, organizeAnime, uploadAnimeCover } from "@/api/endpoints"
import type { Anime, EpisodeStreamVariant } from "@/api/types"

function clearUpscaledStorageKey(animes: Anime[] | undefined, episodeId: string): Anime[] {
  return (animes ?? []).map((anime) => ({
    ...anime,
    seasons: anime.seasons.map((season) => ({
      ...season,
      episodes: season.episodes.map((episode) => {
        if (episode.id !== episodeId) return episode
        return { ...episode, upscaledStorageKey: undefined }
      }),
    })),
  }))
}

export function useCatalog() {
  return useQuery({
    queryKey: ["catalog"],
    queryFn: fetchCatalog,
    staleTime: 30 * 1000,
    meta: { silentError: true },
  })
}

export function useDeleteAnime() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteAnime,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["catalog"] })
      toast.success("Anime removido do catálogo")
    },
  })
}

export function useDeleteEpisode() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteEpisode,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["catalog"] })
      toast.success("Episódio removido")
    },
  })
}

export function useDeleteUpscaledEpisode() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteUpscaledEpisode,
    onSuccess: (_, episodeId) => {
      queryClient.setQueryData<Anime[]>(["catalog"], (animes) => clearUpscaledStorageKey(animes, episodeId))
      queryClient.invalidateQueries({ queryKey: ["catalog"] })
      toast.success("Versão upscale removida")
    },
  })
}

export function useOrganizeAnime() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: organizeAnime,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["catalog"] })
      toast.success("Episódios organizados")
    },
  })
}

export function useUploadCover() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ animeId, file }: { animeId: string; file: File }) =>
      uploadAnimeCover(animeId, file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["catalog"] })
      toast.success("Capa atualizada")
    },
  })
}

export function useEpisodeStream(episodeId: string | null, variant: EpisodeStreamVariant) {
  return useQuery({
    queryKey: ["episode-stream", episodeId, variant],
    queryFn: () => fetchEpisodeStreamURL(episodeId!, variant),
    enabled: episodeId !== null,
    staleTime: 50 * 60 * 1000,
  })
}
