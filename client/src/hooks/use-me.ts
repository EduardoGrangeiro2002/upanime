import { useQuery } from "@tanstack/react-query"
import { fetchMe } from "@/api/endpoints"

export function useMe() {
  return useQuery({
    queryKey: ["me"],
    queryFn: fetchMe,
    staleTime: 5 * 60 * 1000,
    meta: { silentError: true },
  })
}
