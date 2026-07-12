import { cn } from "@/lib/utils"

interface ProgressBarProps {
  progress: number
  className?: string
}

export function ProgressBar({ progress, className }: ProgressBarProps) {
  const clampedProgress = Math.min(100, Math.max(0, progress))

  return (
    <div className={cn("h-2 w-full overflow-hidden rounded-full bg-surface-highest", className)}>
      <div
        className="h-full rounded-full bg-gradient-to-r from-primary to-primary-dim shadow-[0_0_8px_rgba(255,92,146,0.3)] transition-all duration-300 ease-out"
        style={{ width: `${clampedProgress}%` }}
        role="progressbar"
        aria-valuenow={clampedProgress}
        aria-valuemin={0}
        aria-valuemax={100}
      />
    </div>
  )
}
