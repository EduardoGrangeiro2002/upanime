import { describe, expect, it } from "vitest"
import { resumeTime, progressPct, buildProgressMap } from "../../src/hooks/use-playback-progress"
import type { WatchProgressItem } from "../../src/api/types"

function item(episodeId: string, position: number, duration: number): WatchProgressItem {
  return {
    episodeId,
    animeId: "a1",
    animeTitle: "Anime",
    animeImageUrl: "",
    episodeTitle: "Ep",
    episodeNumber: "1",
    seasonNumber: 1,
    position,
    duration,
    updatedAt: "2026-07-20T10:00:00Z",
  }
}

describe("resumeTime", () => {
  it("starts from zero when position is at most 5 seconds", () => {
    expect(resumeTime(0, 1400)).toBe(0)
    expect(resumeTime(5, 1400)).toBe(0)
  })

  it("resumes from the saved position mid-episode", () => {
    expect(resumeTime(120, 1400)).toBe(120)
  })

  it("starts from zero when the episode is nearly finished", () => {
    expect(resumeTime(1330, 1400)).toBe(0)
    expect(resumeTime(1400, 1400)).toBe(0)
  })

  it("resumes past 5 seconds when duration is unknown", () => {
    expect(resumeTime(120, 0)).toBe(120)
  })
})

describe("progressPct", () => {
  it("returns 0 without position or duration", () => {
    expect(progressPct(0, 1400)).toBe(0)
    expect(progressPct(120, 0)).toBe(0)
  })

  it("computes the rounded percentage", () => {
    expect(progressPct(350, 1400)).toBe(25)
  })

  it("caps at 100", () => {
    expect(progressPct(2000, 1400)).toBe(100)
  })
})

describe("buildProgressMap", () => {
  it("indexes items by episode id", () => {
    const map = buildProgressMap([item("ep-1", 120, 1400), item("ep-2", 60, 1400)])
    expect(map["ep-1"].position).toBe(120)
    expect(map["ep-2"].position).toBe(60)
  })

  it("returns an empty map for undefined", () => {
    expect(buildProgressMap(undefined)).toEqual({})
  })
})
