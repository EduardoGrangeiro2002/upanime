import { describe, it, expect, beforeEach } from "vitest"
import { getProgress, getProgressPct, clearProgress } from "../../src/hooks/use-playback-progress"

const STORAGE_KEY = "upanime:playback-progress"

beforeEach(() => {
  localStorage.clear()
})

describe("getProgress", () => {
  it("returns 0 when no progress is stored", () => {
    expect(getProgress("ep-1")).toBe(0)
  })

  it("returns the stored time for an episode", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ "ep-1": 120 }))
    expect(getProgress("ep-1")).toBe(120)
  })

  it("returns 0 for a different episode", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ "ep-1": 120 }))
    expect(getProgress("ep-2")).toBe(0)
  })

  it("returns 0 when localStorage contains invalid JSON", () => {
    localStorage.setItem(STORAGE_KEY, "not json")
    expect(getProgress("ep-1")).toBe(0)
  })
})

describe("getProgress with duration entries", () => {
  it("returns the stored time from an entry with duration", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ "ep-1": { t: 300, d: 1440 } }))
    expect(getProgress("ep-1")).toBe(300)
  })
})

describe("getProgressPct", () => {
  it("returns 0 when nothing is stored", () => {
    expect(getProgressPct("ep-1")).toBe(0)
  })

  it("computes percentage from time and duration", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ "ep-1": { t: 360, d: 1440 } }))
    expect(getProgressPct("ep-1")).toBe(25)
  })

  it("caps at 100", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ "ep-1": { t: 2000, d: 1440 } }))
    expect(getProgressPct("ep-1")).toBe(100)
  })

  it("returns 0 for legacy entries without duration instead of guessing", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ "ep-1": 500 }))
    expect(getProgressPct("ep-1")).toBe(0)
  })
})

describe("clearProgress", () => {
  it("removes progress for a specific episode", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ "ep-1": 120, "ep-2": 60 }))
    clearProgress("ep-1")
    expect(getProgress("ep-1")).toBe(0)
    expect(getProgress("ep-2")).toBe(60)
  })

  it("does nothing when episode has no progress", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ "ep-1": 120 }))
    clearProgress("ep-999")
    expect(getProgress("ep-1")).toBe(120)
  })
})
