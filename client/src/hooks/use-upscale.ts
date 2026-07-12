import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { fetchUpscaleJobs, startUpscale, deleteUpscaleJob } from "@/api/endpoints"
import type { UpscaleJob, UpscaleRequest } from "@/api/types"

function hasActiveJobs(jobs: UpscaleJob[] | undefined): boolean {
  return (jobs ?? []).some(
    (j) => j.status === "queued" || j.status === "processing"
  )
}

export function useUpscalePolling() {
  return useQuery({
    queryKey: ["upscale-jobs"],
    queryFn: fetchUpscaleJobs,
    staleTime: 10 * 1000,
    refetchInterval: (query) => {
      return hasActiveJobs(query.state.data) ? 5000 : false
    },
  })
}

export function useStartUpscale() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (req: UpscaleRequest) => startUpscale(req),
    onSuccess: (_, req) => {
      queryClient.invalidateQueries({ queryKey: ["upscale-jobs"] })
      toast.success(
        req.episodeIds.length === 1
          ? "Upscale iniciado para 1 episódio"
          : `Upscale iniciado para ${req.episodeIds.length} episódios`,
      )
    },
  })
}

export function useDeleteUpscaleJob() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteUpscaleJob,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["upscale-jobs"] })
      toast.success("Job de upscale cancelado")
    },
  })
}
