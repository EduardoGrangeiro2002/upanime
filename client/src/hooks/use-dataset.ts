import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { fetchDatasetQueue, fetchDatasetStats, submitDatasetVerdict } from "@/api/endpoints"
import type { DatasetSample, DatasetVerdict } from "@/api/types"

export function useDatasetQueue() {
  return useQuery({
    queryKey: ["dataset-queue"],
    queryFn: () => fetchDatasetQueue(),
  })
}

export function useDatasetStats() {
  return useQuery({
    queryKey: ["dataset-stats"],
    queryFn: fetchDatasetStats,
  })
}

export function useSubmitVerdict() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, verdict }: { id: string; verdict: DatasetVerdict }) => submitDatasetVerdict(id, verdict),
    onSuccess: (_, { id }) => {
      const remaining = (queryClient.getQueryData<DatasetSample[]>(["dataset-queue"]) ?? []).filter(
        (sample) => sample.id !== id,
      )
      queryClient.setQueryData(["dataset-queue"], remaining)
      queryClient.invalidateQueries({ queryKey: ["dataset-stats"] })
      if (remaining.length === 0) {
        queryClient.invalidateQueries({ queryKey: ["dataset-queue"] })
      }
    },
  })
}
