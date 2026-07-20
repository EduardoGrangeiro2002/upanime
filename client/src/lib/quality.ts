import type { Episode } from "@/api/types"

export interface QualityOption {
  label: string
  variant: string
}

export function heightLabel(height: number): string {
  if (height >= 2160) return "4K"
  return `${height}p`
}

export function buildQualityOptions(episode: Episode): QualityOption[] {
  const options: QualityOption[] = []

  if (episode.storageKey) {
    options.push({ label: "Original", variant: "original" })
  }

  const variants = episode.upscaledVariants ?? []
  if (variants.length > 0) {
    for (const variant of [...variants].sort((a, b) => b.height - a.height)) {
      options.push({ label: heightLabel(variant.height), variant: `${variant.height}p` })
    }
    return options
  }

  if (episode.upscaledStorageKey) {
    options.push({ label: "Upscale", variant: "upscaled" })
  }
  return options
}
