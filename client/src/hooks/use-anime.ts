import { useQuery } from "@tanstack/react-query"
import { fetchAnime } from "@/api/endpoints"
import type { Anime } from "@/api/types"

export function useAnime(url: string) {
  return useQuery<Anime>({
    queryKey: ["anime", url],
    queryFn: () => fetchAnime(url),
    enabled: url.length > 0,
    retry: false,
    staleTime: 5 * 60 * 1000,
  })
}
