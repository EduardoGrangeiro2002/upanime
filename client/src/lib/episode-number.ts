export function detectEpisodeNumber(filename: string, fallback: number): string {
  const base = filename.replace(/\.[^.]+$/, "")
  const matches = base.match(/\d+/g)
  if (!matches || matches.length === 0) return String(fallback)
  return String(parseInt(matches[matches.length - 1], 10))
}
