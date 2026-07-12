export const AGGRESSIVE_UPSCALE_RATIO = 3

export function isAggressiveUpscale(sourceHeight: number, targetHeight: number): boolean {
  if (sourceHeight <= 0) return false
  return targetHeight / sourceHeight > AGGRESSIVE_UPSCALE_RATIO
}

export function maxSafeTargetHeight(sourceHeight: number, targets: readonly number[]): number | null {
  const safe = targets.filter((target) => !isAggressiveUpscale(sourceHeight, target))
  if (safe.length === 0) return null
  return Math.max(...safe)
}
