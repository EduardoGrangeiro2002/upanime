export type EpisodeType = "episode" | "movie" | "ova"

export type DownloadStatus = "queued" | "resolving" | "downloading" | "completed" | "failed"

export interface Anime {
  id: string
  title: string
  url: string
  imageUrl: string
  description: string
  coverPath?: string
  coverUrl?: string
  genres?: string[]
  seasons: Season[]
}

export interface UploadEpisodeParams {
  animeTitle: string
  seasonNumber: number
  episodeNumber: string
  file: File
}

export interface UploadEpisodeResponse {
  animeId: string
  episode: Episode
  replaced: boolean
}

export interface Season {
  number: number
  label: string
  type: EpisodeType
  episodes: Episode[]
}

export interface EpisodeVariant {
  height: number
  storageKey: string
}

export interface Episode {
  id: string
  title: string
  number: string
  seasonNumber: number
  type: EpisodeType
  url: string
  storageKey?: string
  upscaledStorageKey?: string
  upscaledVariants?: EpisodeVariant[]
}

export interface EpisodeProgress {
  position: number
  duration: number
  updatedAt: string
}

export interface WatchProgressItem extends EpisodeProgress {
  episodeId: string
  animeId: string
  animeTitle: string
  animeImageUrl: string
  episodeTitle: string
  episodeNumber: string
  seasonNumber: number
}

export interface Download {
  id: string
  animeId: string
  animeTitle: string
  animeImageUrl: string
  episodeId: string
  episodeTitle: string
  seasonNumber: number
  episodeNumber: string
  status: DownloadStatus
  progress: number
  speed: string
  eta: string
  error?: string
}

export interface DownloadEpisodeInput {
  title: string
  number: string
  url: string
  seasonNumber: number
}

export interface DownloadRequest {
  animeId?: string
  animeTitle?: string
  animeImageUrl: string
  description?: string
  sourceUrl: string
  seasonNumber?: number
  episodes: DownloadEpisodeInput[]
}

export interface ProgressEvent {
  downloadId: string
  progress: number
  speed: string
  eta: string
  status: DownloadStatus
}

export type UpscaleStatus = "queued" | "processing" | "completed" | "failed"

export type AuthStep = "change_password" | "mfa" | "ok"

export interface Me {
  email: string
  isAdmin: boolean
}

export interface UserSummary {
  email: string
  isAdmin: boolean
  pending: boolean
}

export type UpscaleJobType = "upscale"
export type TargetHeight = 1080 | 1440 | 2160
export type EpisodeStreamVariant = "original" | "upscaled"

export interface UpscaleJob {
  id: string
  episodeId: string
  animeId: string
  type: UpscaleJobType
  targetHeight?: TargetHeight
  animeTitle: string
  animeImageUrl: string
  episodeTitle: string
  episodeNumber: string
  seasonNumber: number
  resultStorageKey?: string
  status: UpscaleStatus
  error?: string
}

export interface EncodeParams {
  batchSize: number
  sharpen: number
  saturation: number
  contrast: number
}

export type DatasetVerdict = "approved" | "rejected" | "needs_edit"

export interface DatasetSample {
  id: string
  source: string
  class: string
  frameUrl: string
  maskUrl: string
  animeTitle: string
  episode: string
  timestampS: number
  teacherProb: number
  status: string
  createdAt: string
}

export interface DatasetClassStat {
  class: string
  status: string
  count: number
}

export interface DatasetStats {
  total: number
  pending: number
  approved: number
  rejected: number
  needsEdit: number
  byClass: DatasetClassStat[]
}

export interface UpscaleRequest {
  animeId: string
  episodeIds: string[]
  targetHeight: TargetHeight
  batchSize?: number
  sharpen?: number
  saturation?: number
  contrast?: number
  interpolate?: boolean
  panRatio?: number
  effects?: boolean
  effectsStrength?: number
  effectsSensitivity?: number
  skipUpscale?: boolean
}
