import { useMemo, useState } from "react"
import { Wand2, Library, Settings, ListOrdered, Clock, X, Loader2, TriangleAlert, Sparkles } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useCatalog } from "@/hooks/use-catalog"
import { useEpisodeHeights } from "@/hooks/use-episode-heights"
import { useUpscalePolling, useStartUpscale, useDeleteUpscaleJob } from "@/hooks/use-upscale"
import { isAggressiveUpscale, maxSafeTargetHeight } from "@/lib/upscale-warning"

import type { Anime, EncodeParams, Episode, TargetHeight, UpscaleJob } from "@/api/types"
import { cn } from "@/lib/utils"

const TARGET_HEIGHTS: readonly TargetHeight[] = [1080, 1440, 2160]

export function EditionPage() {
  const [selectedAnimeId, setSelectedAnimeId] = useState<string | null>(null)
  const [selectedEpisodeIds, setSelectedEpisodeIds] = useState<Set<string>>(new Set())
  const [targetHeight, setTargetHeight] = useState<TargetHeight>(1080)
  const [interpolate, setInterpolate] = useState(false)
  const [panRatio, setPanRatio] = useState(0.6)
  const [effects, setEffects] = useState(false)
  const [effectsParams, setEffectsParams] = useState({ strength: 1.0, sensitivity: 1.0 })
  const [skipUpscale, setSkipUpscale] = useState(false)
  const [upscaler, setUpscaler] = useState<"compact" | "apisr">("compact")
  const [encodeParams, setEncodeParams] = useState<EncodeParams>({ batchSize: 2, sharpen: 0.5, saturation: 1.20, contrast: 1.05 })
  const startUpscale = useStartUpscale()

  const { data: animes } = useCatalog()
  const selectedEpisodes = useMemo(() => {
    const anime = animes?.find((item) => item.id === selectedAnimeId)
    if (!anime) return []
    return anime.seasons
      .flatMap((season) => season.episodes)
      .filter((episode) => selectedEpisodeIds.has(episode.id))
  }, [animes, selectedAnimeId, selectedEpisodeIds])
  const episodeHeights = useEpisodeHeights(selectedEpisodes.map((episode) => episode.id))

  const handleStart = () => {
    if (!selectedAnimeId || selectedEpisodeIds.size === 0) return
    startUpscale.mutate(
      {
        animeId: selectedAnimeId,
        episodeIds: Array.from(selectedEpisodeIds),
        targetHeight,
        batchSize: encodeParams.batchSize,
        sharpen: encodeParams.sharpen,
        saturation: encodeParams.saturation,
        contrast: encodeParams.contrast,
        interpolate,
        ...(interpolate ? { panRatio } : {}),
        effects,
        ...(effects ? { effectsStrength: effectsParams.strength, effectsSensitivity: effectsParams.sensitivity, skipUpscale } : {}),
        ...(upscaler !== "compact" ? { upscaler } : {}),
      },
      {
        onSuccess: () => {
          setSelectedEpisodeIds(new Set())
          setSelectedAnimeId(null)
        },
      },
    )
  }

  return (
    <div className="space-y-8 px-4 py-4 md:px-8 md:py-8 max-w-5xl mx-auto pb-28">
      <HeroSection targetHeight={targetHeight} />
      <CloudLibrary
        selectedAnimeId={selectedAnimeId}
        selectedEpisodeIds={selectedEpisodeIds}
        onSelectAnime={setSelectedAnimeId}
        onToggleEpisode={(id) => {
          setSelectedEpisodeIds((prev) => {
            const next = new Set(prev)
            if (next.has(id)) {
              next.delete(id)
              return next
            }
            next.add(id)
            return next
          })
        }}
        onSelectAll={(ids) => setSelectedEpisodeIds(new Set(ids))}
        onDeselectAll={() => setSelectedEpisodeIds(new Set())}
      />
      <AggressiveUpscaleWarning
        episodes={selectedEpisodes}
        heights={episodeHeights}
        targetHeight={targetHeight}
      />
      <ProcessingConfig
        targetHeight={targetHeight}
        onChange={setTargetHeight}
        interpolate={interpolate}
        onInterpolateChange={setInterpolate}
        panRatio={panRatio}
        onPanRatioChange={setPanRatio}
        effects={effects}
        onEffectsChange={setEffects}
        effectsParams={effectsParams}
        onEffectsParamsChange={setEffectsParams}
        skipUpscale={skipUpscale}
        onSkipUpscaleChange={setSkipUpscale}
        upscaler={upscaler}
        onUpscalerChange={setUpscaler}
        encodeParams={encodeParams}
        onEncodeParamsChange={setEncodeParams}
      />
      <ProcessingQueue />
      <RecentHistory />

      {selectedEpisodeIds.size > 0 && (
        <div className="fixed bottom-24 md:bottom-4 left-1/2 -translate-x-1/2 z-30 flex max-w-[calc(100vw-2rem)] items-center gap-3 rounded-2xl glass border border-white/[0.08] px-4 py-3 shadow-2xl animate-in fade-in slide-in-from-bottom-2 duration-200">
          <span className="text-sm text-muted-foreground whitespace-nowrap">
            {selectedEpisodeIds.size} episódio{selectedEpisodeIds.size !== 1 ? "s" : ""} · {targetHeight}p{interpolate ? " · 60fps" : ""}{upscaler === "apisr" ? " · APISR" : ""}
          </span>
          <Button
            variant="gradient"
            size="sm"
            disabled={startUpscale.isPending}
            onClick={handleStart}
          >
            {startUpscale.isPending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Wand2 className="h-4 w-4" aria-hidden="true" />}
            Iniciar Upscale
          </Button>
        </div>
      )}
    </div>
  )
}

function HeroSection({ targetHeight }: { targetHeight: TargetHeight }) {
  return (
    <div className="relative overflow-hidden rounded-2xl bg-surface-high p-4 md:p-8">
      <div className="absolute top-4 right-4 opacity-10" aria-hidden="true">
        <Wand2 className="h-32 w-32 text-primary" />
      </div>
      <div className="relative z-10 max-w-lg space-y-3">
        <h1 className="font-display text-2xl md:text-4xl font-bold tracking-tighter">
          Upscale
        </h1>
        <p className="text-muted-foreground text-sm leading-relaxed">
          {`Pipeline GPU com Real-ESRGAN AnimeVideo v3: os episódios selecionados são reprocessados e salvos em ${targetHeight}p.`}
        </p>
        <p className="text-xs text-muted-foreground/80">
          Selecione um anime e os episódios abaixo para começar.
        </p>
      </div>
    </div>
  )
}

function CloudLibrary({
  selectedAnimeId,
  selectedEpisodeIds,
  onSelectAnime,
  onToggleEpisode,
  onSelectAll,
  onDeselectAll,
}: {
  selectedAnimeId: string | null
  selectedEpisodeIds: Set<string>
  onSelectAnime: (id: string | null) => void
  onToggleEpisode: (id: string) => void
  onSelectAll: (ids: string[]) => void
  onDeselectAll: () => void
}) {
  const { data: animes } = useCatalog()

  const selectedAnime = animes?.find((anime) => anime.id === selectedAnimeId)
  const downloadedEpisodes = selectedAnime
    ? selectedAnime.seasons.flatMap((season) => season.episodes.filter((episode) => episode.storageKey))
    : []

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Library className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
        <h2 className="font-display text-xl font-bold">Biblioteca</h2>
      </div>

      {!animes || animes.length === 0 ? (
        <div className="rounded-xl bg-surface p-8 text-center opacity-60">
          <Library className="h-10 w-10 mx-auto mb-3 text-muted-foreground opacity-40" aria-hidden="true" />
          <p className="text-sm text-muted-foreground">Nenhum anime no catálogo</p>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="flex gap-3 overflow-x-auto scrollbar-thin pb-2">
            {animes.map((anime) => (
              <AnimeCard
                key={anime.id}
                anime={anime}
                isSelected={anime.id === selectedAnimeId}
                onSelect={() => {
                  if (anime.id === selectedAnimeId) {
                    onSelectAnime(null)
                    onDeselectAll()
                    return
                  }
                  onSelectAnime(anime.id)
                  onDeselectAll()
                }}
              />
            ))}
          </div>

          {selectedAnime && downloadedEpisodes.length > 0 && (
            <EpisodeSelector
              episodes={downloadedEpisodes}
              selectedIds={selectedEpisodeIds}
              onToggle={onToggleEpisode}
              onSelectAll={() => onSelectAll(downloadedEpisodes.map((episode) => episode.id))}
              onDeselectAll={onDeselectAll}
            />
          )}

          {selectedAnime && downloadedEpisodes.length === 0 && (
            <div className="rounded-xl bg-surface p-6 text-center">
              <p className="text-sm text-muted-foreground">Nenhum episódio baixado neste anime</p>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function AnimeCard({
  anime,
  isSelected,
  onSelect,
}: {
  anime: Anime
  isSelected: boolean
  onSelect: () => void
}) {
  const downloadedCount = anime.seasons.reduce(
    (accumulator, season) => accumulator + season.episodes.filter((episode) => episode.storageKey).length,
    0,
  )

  return (
    <button
      onClick={onSelect}
      aria-pressed={isSelected}
      className={cn(
        "shrink-0 w-[140px] rounded-xl overflow-hidden bg-surface transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        isSelected && "ring-2 ring-primary shadow-[0_0_12px_rgba(255,92,146,0.3)]",
        !isSelected && "hover:ring-1 hover:ring-muted-foreground/30",
      )}
    >
      <div className="aspect-[2/3] bg-surface-high relative">
        {(anime.coverUrl || anime.imageUrl) ? (
          <img
            src={anime.coverUrl || anime.imageUrl}
            alt=""
            className="h-full w-full object-cover"
          />
        ) : (
          <div className="flex h-full items-center justify-center">
            <span className="font-display text-lg font-bold text-muted-foreground">{anime.title.charAt(0)}</span>
          </div>
        )}
      </div>
      <div className="p-2">
        <p className="text-xs font-medium truncate" title={anime.title}>{anime.title}</p>
        <p className="text-[10px] text-muted-foreground">{downloadedCount} episódios</p>
      </div>
    </button>
  )
}

function EpisodeSelector({
  episodes,
  selectedIds,
  onToggle,
  onSelectAll,
  onDeselectAll,
}: {
  episodes: Episode[]
  selectedIds: Set<string>
  onToggle: (id: string) => void
  onSelectAll: () => void
  onDeselectAll: () => void
}) {
  const allSelected = episodes.every((episode) => selectedIds.has(episode.id))

  return (
    <div className="rounded-xl bg-surface p-4 space-y-3">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium">{episodes.length} episódios disponíveis</span>
        <button
          onClick={allSelected ? onDeselectAll : onSelectAll}
          className="text-xs text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
        >
          {allSelected ? "Desmarcar todos" : "Selecionar todos"}
        </button>
      </div>
      <div className="max-h-72 overflow-y-auto space-y-1 scrollbar-thin pr-1">
        {episodes.map((episode) => (
          <label
            key={episode.id}
            className="flex items-center gap-3 rounded-lg px-3 py-2 hover:bg-surface-high cursor-pointer"
          >
            <input
              type="checkbox"
              checked={selectedIds.has(episode.id)}
              onChange={() => onToggle(episode.id)}
              className="accent-primary rounded-sm"
            />
            <span className="text-xs text-muted-foreground w-10 tabular-nums">
              {episode.seasonNumber > 0 ? `${episode.seasonNumber}x${episode.number.padStart(2, "0")}` : ""}
            </span>
            <span className="text-sm truncate" title={episode.title}>{episode.title}</span>
            {episode.upscaledStorageKey && (
              <Badge variant="success" className="ml-auto shrink-0 gap-1 text-[10px]">
                <Sparkles className="h-3 w-3" aria-hidden="true" />
                Upscalado
              </Badge>
            )}
          </label>
        ))}
      </div>
    </div>
  )
}

export function AggressiveUpscaleWarning({
  episodes,
  heights,
  targetHeight,
}: {
  episodes: Episode[]
  heights: Map<string, number>
  targetHeight: TargetHeight
}) {
  const risky = episodes.filter((episode) => {
    const sourceHeight = heights.get(episode.id)
    if (sourceHeight === undefined) return false
    return isAggressiveUpscale(sourceHeight, targetHeight)
  })

  if (risky.length === 0) return null

  const lowestSource = Math.min(...risky.map((episode) => heights.get(episode.id)!))
  const suggestion = maxSafeTargetHeight(lowestSource, TARGET_HEIGHTS)

  return (
    <div
      role="status"
      className="flex items-start gap-3 rounded-xl border border-amber-500/30 bg-amber-500/10 p-4"
    >
      <TriangleAlert className="h-5 w-5 shrink-0 text-amber-500" aria-hidden="true" />
      <div className="space-y-1">
        <p className="text-sm font-medium text-amber-500">Upscale agressivo demais para a fonte</p>
        <p className="text-xs text-muted-foreground">
          {risky.length === 1
            ? `1 episódio selecionado tem fonte de ~${lowestSource} linhas`
            : `${risky.length} episódios selecionados têm fonte de até ~${lowestSource} linhas`}
          {` — em ${targetHeight}p o modelo teria que inventar a maior parte dos pixels. `}
          {suggestion ? `Recomendado: ${suggestion}p.` : "Recomendado manter uma resolução menor."}
        </p>
      </div>
    </div>
  )
}

export function ProcessingConfig({
  targetHeight,
  onChange,
  interpolate,
  onInterpolateChange,
  panRatio,
  onPanRatioChange,
  effects,
  onEffectsChange,
  effectsParams,
  onEffectsParamsChange,
  skipUpscale,
  onSkipUpscaleChange,
  upscaler,
  onUpscalerChange,
  encodeParams,
  onEncodeParamsChange,
}: {
  targetHeight: TargetHeight
  onChange: (height: TargetHeight) => void
  interpolate: boolean
  onInterpolateChange: (value: boolean) => void
  panRatio: number
  onPanRatioChange: (value: number) => void
  effects: boolean
  onEffectsChange: (value: boolean) => void
  effectsParams: { strength: number; sensitivity: number }
  onEffectsParamsChange: (params: { strength: number; sensitivity: number }) => void
  skipUpscale: boolean
  onSkipUpscaleChange: (value: boolean) => void
  upscaler: "compact" | "apisr"
  onUpscalerChange: (value: "compact" | "apisr") => void
  encodeParams: EncodeParams
  onEncodeParamsChange: (params: EncodeParams) => void
}) {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Settings className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
        <h2 className="font-display text-xl font-bold">Processamento</h2>
      </div>

      <div className="rounded-xl glass p-4 md:p-6 space-y-4">
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">Resolução de saída</span>
            <Badge variant="secondary" className="text-[10px]">
              {targetHeight}p
            </Badge>
          </div>
          <div className="flex gap-2" role="group" aria-label="Resolução de saída">
            <Button
              type="button"
              size="sm"
              variant={targetHeight === 1080 ? "default" : "outline"}
              aria-pressed={targetHeight === 1080}
              onClick={() => onChange(1080)}
            >
              Full HD
            </Button>
            <Button
              type="button"
              size="sm"
              variant={targetHeight === 1440 ? "default" : "outline"}
              aria-pressed={targetHeight === 1440}
              onClick={() => onChange(1440)}
            >
              2K
            </Button>
            <Button
              type="button"
              size="sm"
              variant={targetHeight === 2160 ? "default" : "outline"}
              aria-pressed={targetHeight === 2160}
              onClick={() => onChange(2160)}
            >
              4K
            </Button>
          </div>
        </div>
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span
              className="text-sm font-medium underline decoration-dotted decoration-muted-foreground/40 underline-offset-4 cursor-help"
              data-tooltip="APISR (transformer): linhas mais nítidas e limpeza de artefatos em fontes antigas de baixa resolução (360p/480p). ~5× mais lento que o padrão; em fontes 1080p pode estourar o tempo limite do job."
              tabIndex={0}
            >
              Upscaler APISR (fontes antigas)
            </span>
            <Button
              type="button"
              size="sm"
              variant={upscaler === "apisr" ? "default" : "outline"}
              aria-pressed={upscaler === "apisr"}
              onClick={() => onUpscalerChange(upscaler === "apisr" ? "compact" : "apisr")}
            >
              {upscaler === "apisr" ? "Ativado" : "Desativado"}
            </Button>
          </div>
        </div>
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span
              className="text-sm font-medium underline decoration-dotted decoration-muted-foreground/40 underline-offset-4 cursor-help"
              data-tooltip="Interpolação seletiva com RIFE: pans de câmera ganham frames até 60fps; cenas de ação, flashes e cortes preservam o timing original."
              tabIndex={0}
            >
              Interpolação 60 FPS (RIFE seletivo)
            </span>
            <Button
              type="button"
              size="sm"
              variant={interpolate ? "default" : "outline"}
              aria-pressed={interpolate}
              onClick={() => onInterpolateChange(!interpolate)}
            >
              {interpolate ? "Ativada" : "Desativada"}
            </Button>
          </div>
          {interpolate && (
            <ParamSlider
              label="Sensibilidade de pan"
              tooltip="Quanto maior, mais movimentos de câmera são aceitos para interpolação (tolera mais parallax). Valores altos podem suavizar cenas que deviam ficar secas."
              value={panRatio}
              min={0.6}
              max={0.9}
              step={0.05}
              onChange={onPanRatioChange}
            />
          )}
        </div>
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span
              className="text-sm font-medium underline decoration-dotted decoration-muted-foreground/40 underline-offset-4 cursor-help"
              data-tooltip="Composição moderna nos efeitos de energia (fogo, explosão, magia): bloom, cor, luz na cena e textura — só em planos onde a IA detecta efeito; o resto passa intocado."
              tabIndex={0}
            >
              Modernizar efeitos
            </span>
            <Button
              type="button"
              size="sm"
              variant={effects ? "default" : "outline"}
              aria-pressed={effects}
              onClick={() => onEffectsChange(!effects)}
            >
              {effects ? "Ativada" : "Desativada"}
            </Button>
          </div>
          {effects && (
            <>
              <ParamSlider
                label="Intensidade dos efeitos"
                tooltip="Ganho geral da composição (bloom, luz, textura). 1.00 é o calibrado; acima disso o efeito fica mais dramático."
                value={effectsParams.strength}
                min={0.2}
                max={1.5}
                step={0.05}
                onChange={(v) => onEffectsParamsChange({ ...effectsParams, strength: v })}
              />
              <ParamSlider
                label="Sensibilidade da detecção"
                tooltip="Quão agressiva é a detecção de regiões de efeito. Valores altos pegam efeitos mais escuros, mas podem vazar para roupas e cabelos coloridos."
                value={effectsParams.sensitivity}
                min={0.5}
                max={1.5}
                step={0.05}
                onChange={(v) => onEffectsParamsChange({ ...effectsParams, sensitivity: v })}
              />
              <div className="flex items-center justify-between">
                <span
                  className="text-sm font-medium underline decoration-dotted decoration-muted-foreground/40 underline-offset-4 cursor-help"
                  data-tooltip="Modo debug: pula o upscale (Real-ESRGAN) e renderiza na resolução original só com a composição de efeitos — rápido e barato para conferir onde a detecção dispara."
                  tabIndex={0}
                >
                  Prévia sem upscale
                </span>
                <Button
                  type="button"
                  size="sm"
                  variant={skipUpscale ? "default" : "outline"}
                  aria-pressed={skipUpscale}
                  onClick={() => onSkipUpscaleChange(!skipUpscale)}
                >
                  {skipUpscale ? "Ativada" : "Desativada"}
                </Button>
              </div>
            </>
          )}
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <PipelineInfo label="Modelo" value="Real-ESRGAN AnimeVideo v3" />
          <PipelineInfo label="Codificação" value="H.264 medium" />
        </div>
        <div className="space-y-3 pt-2">
          <ParamSlider
            label="Batch"
            tooltip="Quantos frames a GPU processa em paralelo. Valores maiores aceleram o job, mas consomem mais VRAM."
            value={encodeParams.batchSize}
            min={1}
            max={16}
            step={1}
            onChange={(v) => onEncodeParamsChange({ ...encodeParams, batchSize: v })}
          />
          <ParamSlider
            label="Nitidez"
            tooltip="Realce de contornos aplicado depois do upscale. 0 desliga; acima de 1.0 pode criar halos nos traços."
            value={encodeParams.sharpen}
            min={0}
            max={2}
            step={0.1}
            onChange={(v) => onEncodeParamsChange({ ...encodeParams, sharpen: v })}
          />
          <ParamSlider
            label="Saturação"
            tooltip="Intensidade das cores no vídeo final. 1.00 mantém as cores originais."
            value={encodeParams.saturation}
            min={0.5}
            max={2}
            step={0.05}
            onChange={(v) => onEncodeParamsChange({ ...encodeParams, saturation: v })}
          />
          <ParamSlider
            label="Contraste"
            tooltip="Contraste do vídeo final. 1.00 mantém o original."
            value={encodeParams.contrast}
            min={0.5}
            max={2}
            step={0.05}
            onChange={(v) => onEncodeParamsChange({ ...encodeParams, contrast: v })}
          />
        </div>
      </div>
    </div>
  )
}

function PipelineInfo({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between rounded-lg bg-surface-high/50 px-4 py-2.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <Badge variant="secondary" className="text-[10px]">{value}</Badge>
    </div>
  )
}

function formatParam(value: number, step: number): string {
  if (step >= 1) return String(Math.round(value))
  return value.toFixed(2)
}

function ParamSlider({
  label,
  tooltip,
  value,
  min,
  max,
  step,
  onChange,
}: {
  label: string
  tooltip: string
  value: number
  min: number
  max: number
  step: number
  onChange: (value: number) => void
}) {
  return (
    <div className="flex items-center gap-4">
      <span
        className="text-xs text-muted-foreground w-20 shrink-0 underline decoration-dotted decoration-muted-foreground/40 underline-offset-4 cursor-help"
        data-tooltip={tooltip}
        tabIndex={0}
      >
        {label}
      </span>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        aria-label={label}
        onChange={(e) => onChange(parseFloat(e.target.value))}
        className="flex-1 h-1.5 py-2.5 box-content bg-clip-content rounded-full appearance-none bg-surface-high accent-primary cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      />
      <Badge variant="secondary" className="text-[10px] w-12 text-center tabular-nums">
        {formatParam(value, step)}
      </Badge>
    </div>
  )
}

function ProcessingQueue() {
  const { data: jobs } = useUpscalePolling()
  const deleteJob = useDeleteUpscaleJob()
  const activeJobs = (jobs ?? []).filter((job) => job.status === "queued" || job.status === "processing")

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <ListOrdered className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
        <h2 className="font-display text-xl font-bold">Fila de Processamento</h2>
        {activeJobs.length > 0 && (
          <Badge variant="secondary" className="text-[10px]">{activeJobs.length}</Badge>
        )}
      </div>

      {activeJobs.length === 0 ? (
        <div className="rounded-xl bg-surface p-8 text-center opacity-60">
          <ListOrdered className="h-10 w-10 mx-auto mb-3 text-muted-foreground opacity-40" aria-hidden="true" />
          <p className="text-sm text-muted-foreground">Nenhum processamento ativo</p>
        </div>
      ) : (
        <div className="max-h-96 overflow-y-auto space-y-2 scrollbar-thin pr-1">
          {activeJobs.map((job) => (
            <UpscaleJobItem
              key={job.id}
              job={job}
              onDelete={() => deleteJob.mutate(job.id)}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function RecentHistory() {
  const { data: jobs } = useUpscalePolling()
  const finishedJobs = (jobs ?? []).filter((job) => job.status === "completed" || job.status === "failed")

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Clock className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
        <h2 className="font-display text-xl font-bold">Histórico Recente</h2>
      </div>

      {finishedJobs.length === 0 ? (
        <div className="rounded-xl bg-surface p-8 text-center opacity-60">
          <Clock className="h-10 w-10 mx-auto mb-3 text-muted-foreground opacity-40" aria-hidden="true" />
          <p className="text-sm text-muted-foreground">Histórico vazio</p>
        </div>
      ) : (
        <div className="max-h-96 overflow-y-auto space-y-2 scrollbar-thin pr-1">
          {finishedJobs.map((job) => (
            <UpscaleJobItem key={job.id} job={job} />
          ))}
        </div>
      )}
    </div>
  )
}

const UPSCALE_STATUS_LABELS: Record<string, string> = {
  queued: "Na fila",
  processing: "Processando",
  completed: "Completo",
  failed: "Falhou",
}

function statusVariant(status: string) {
  if (status === "completed") return "success" as const
  if (status === "failed") return "destructive" as const
  return "secondary" as const
}

function UpscaleJobItem({ job, onDelete }: { job: UpscaleJob; onDelete?: () => void }) {
  const [confirmCancel, setConfirmCancel] = useState(false)
  const isActive = job.status === "queued" || job.status === "processing"

  return (
    <div className={cn(
      "flex items-center gap-3 rounded-xl bg-surface p-3",
      job.status === "completed" && "opacity-60",
    )}>
      {job.animeImageUrl && (
        <img
          src={job.animeImageUrl}
          alt=""
          className="h-10 w-10 rounded-lg object-cover shrink-0"
        />
      )}
      <div className="flex-1 min-w-0 space-y-0.5">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium truncate" title={job.episodeTitle}>{job.episodeTitle}</span>
          {job.targetHeight && (
            <Badge variant="outline" className="text-[10px] px-1.5 py-0">
              {job.targetHeight}p
            </Badge>
          )}
          <Badge variant={statusVariant(job.status)} className="text-[10px] px-1.5 py-0">
            {UPSCALE_STATUS_LABELS[job.status] ?? job.status}
          </Badge>
        </div>
        <div className="flex gap-2 text-xs text-muted-foreground">
          <span>{job.animeTitle}</span>
          {job.seasonNumber > 0 && (
            <span>S{job.seasonNumber}E{job.episodeNumber.padStart(2, "0")}</span>
          )}
        </div>
        {job.error && (
          <p className={cn(
            "text-xs",
            job.status === "failed" ? "text-destructive" : "text-muted-foreground",
          )}>
            {job.error}
          </p>
        )}
      </div>

      {isActive && (
        <div className="flex items-center gap-2 shrink-0">
          <Loader2 className="h-4 w-4 animate-spin text-primary" aria-hidden="true" />
          {onDelete && (
            confirmCancel ? (
              <div className="flex gap-1">
                <Button
                  variant="destructive"
                  size="sm"
                  className="h-7 text-xs"
                  onClick={() => {
                    onDelete()
                    setConfirmCancel(false)
                  }}
                >
                  Cancelar job
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-7 text-xs"
                  onClick={() => setConfirmCancel(false)}
                >
                  Não
                </Button>
              </div>
            ) : (
              <Button
                variant="ghost"
                size="icon"
                aria-label={`Cancelar upscale de ${job.episodeTitle}`}
                data-tooltip="Cancelar upscale"
                data-tooltip-pos="left"
                onClick={() => setConfirmCancel(true)}
                className="h-7 w-7"
              >
                <X className="h-3.5 w-3.5" aria-hidden="true" />
              </Button>
            )
          )}
        </div>
      )}
    </div>
  )
}
