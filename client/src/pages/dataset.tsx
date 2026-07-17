import { useCallback, useEffect, useState } from "react"
import { Check, Database, Eye, EyeOff, Loader2, Pencil, X } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useDatasetQueue, useDatasetStats, useSubmitVerdict } from "@/hooks/use-dataset"
import type { DatasetSample, DatasetVerdict } from "@/api/types"

const CLASS_INFO: Record<string, { label: string; color: string }> = {
  fire: { label: "Fogo", color: "#f97316" },
  lightning: { label: "Relâmpago", color: "#38bdf8" },
  energy: { label: "Energia", color: "#facc15" },
  aura: { label: "Aura", color: "#a78bfa" },
  dark_magic: { label: "Magia escura", color: "#c084fc" },
  beam: { label: "Beam", color: "#f43f5e" },
  none: { label: "Sem efeito", color: "#64748b" },
}

function classInfo(name: string) {
  return CLASS_INFO[name] ?? { label: name, color: "#64748b" }
}

function formatTimestamp(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = seconds - m * 60
  return `${String(m).padStart(2, "0")}:${s.toFixed(1).padStart(4, "0")}`
}

export function DatasetPage() {
  const { data: queue, isLoading } = useDatasetQueue()
  const { data: stats } = useDatasetStats()
  const verdict = useSubmitVerdict()
  const [showMask, setShowMask] = useState(true)

  const current = queue?.[0]

  const submit = useCallback(
    (value: DatasetVerdict) => {
      if (!current || verdict.isPending) return
      verdict.mutate({ id: current.id, verdict: value })
    },
    [current, verdict],
  )

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) return
      const key = event.key.toLowerCase()
      if (key === "a") submit("approved")
      if (key === "r") submit("rejected")
      if (key === "e") submit("needs_edit")
      if (key === "m") setShowMask((value) => !value)
    }
    window.addEventListener("keydown", onKey)
    return () => window.removeEventListener("keydown", onKey)
  }, [submit])

  return (
    <div className="space-y-8 px-4 py-4 md:px-8 md:py-8 max-w-5xl mx-auto pb-28">
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <Database className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
          <h1 className="font-display text-xl font-bold">Triagem do Dataset</h1>
        </div>
        <p className="text-sm text-muted-foreground max-w-2xl">
          Revise as máscaras propostas pelo professor: <strong>A</strong> aprova, <strong>R</strong> rejeita,{" "}
          <strong>E</strong> marca para ajuste manual, <strong>M</strong> mostra/esconde a máscara.
        </p>
        {stats && (
          <div className="flex flex-wrap gap-2">
            <Badge variant="secondary">{stats.pending} pendentes</Badge>
            <Badge variant="secondary" className="text-green-400">{stats.approved} aprovadas</Badge>
            <Badge variant="secondary" className="text-red-400">{stats.rejected} rejeitadas</Badge>
            <Badge variant="secondary" className="text-yellow-400">{stats.needsEdit} para ajuste</Badge>
          </div>
        )}
      </div>

      {isLoading && (
        <div className="rounded-xl glass p-12 flex items-center justify-center gap-3 text-muted-foreground">
          <Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" />
          Carregando fila…
        </div>
      )}

      {!isLoading && !current && (
        <div className="rounded-xl glass p-12 text-center space-y-2">
          <Database className="h-10 w-10 mx-auto text-muted-foreground opacity-40" aria-hidden="true" />
          <p className="text-sm text-muted-foreground">Fila vazia — nenhuma amostra pendente.</p>
          <p className="text-xs text-muted-foreground/70">
            Novas amostras chegam quando o pipeline do professor envia máscaras para revisão.
          </p>
        </div>
      )}

      {current && (
        <TriageCard
          sample={current}
          showMask={showMask}
          onToggleMask={() => setShowMask((value) => !value)}
          onVerdict={submit}
          busy={verdict.isPending}
          remaining={queue?.length ?? 0}
        />
      )}
    </div>
  )
}

function TriageCard({
  sample,
  showMask,
  onToggleMask,
  onVerdict,
  busy,
  remaining,
}: {
  sample: DatasetSample
  showMask: boolean
  onToggleMask: () => void
  onVerdict: (verdict: DatasetVerdict) => void
  busy: boolean
  remaining: number
}) {
  const info = classInfo(sample.class)

  return (
    <div className="rounded-xl glass p-4 md:p-6 space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <Badge style={{ backgroundColor: `${info.color}26`, color: info.color, borderColor: `${info.color}55` }}>
            {info.label}
          </Badge>
          <span className="text-sm text-muted-foreground">
            {sample.animeTitle}
            {sample.episode ? ` · ${sample.episode}` : ""} · {formatTimestamp(sample.timestampS)}
          </span>
          {sample.teacherProb > 0 && (
            <span className="text-xs font-mono text-muted-foreground/80">prob {sample.teacherProb.toFixed(2)}</span>
          )}
        </div>
        <span className="text-xs text-muted-foreground">{remaining} na fila</span>
      </div>

      <div className="relative overflow-hidden rounded-lg bg-black">
        <img src={sample.frameUrl} alt="Frame da amostra" className="w-full select-none" draggable={false} />
        {showMask && (
          <div
            data-testid="mask-overlay"
            aria-hidden="true"
            className="absolute inset-0 pointer-events-none"
            style={{
              backgroundColor: info.color,
              opacity: 0.55,
              WebkitMaskImage: `url(${sample.maskUrl})`,
              maskImage: `url(${sample.maskUrl})`,
              WebkitMaskSize: "100% 100%",
              maskSize: "100% 100%",
            }}
          />
        )}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button onClick={() => onVerdict("approved")} disabled={busy} className="bg-green-600 hover:bg-green-500 text-white">
          <Check className="h-4 w-4" aria-hidden="true" />
          Aprovar (A)
        </Button>
        <Button onClick={() => onVerdict("rejected")} disabled={busy} variant="destructive">
          <X className="h-4 w-4" aria-hidden="true" />
          Rejeitar (R)
        </Button>
        <Button onClick={() => onVerdict("needs_edit")} disabled={busy} variant="outline">
          <Pencil className="h-4 w-4" aria-hidden="true" />
          Ajustar (E)
        </Button>
        <Button onClick={onToggleMask} variant="ghost" className="ml-auto">
          {showMask ? <EyeOff className="h-4 w-4" aria-hidden="true" /> : <Eye className="h-4 w-4" aria-hidden="true" />}
          {showMask ? "Esconder máscara (M)" : "Mostrar máscara (M)"}
        </Button>
      </div>
    </div>
  )
}
