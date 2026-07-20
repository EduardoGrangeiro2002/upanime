import { describe, expect, it } from "vitest"
import { buildQualityOptions, heightLabel } from "../../src/lib/quality"
import type { Episode } from "../../src/api/types"

function episode(partial: Partial<Episode>): Episode {
  return {
    id: "ep-1",
    title: "Ep 1",
    number: "1",
    seasonNumber: 1,
    type: "episode",
    url: "https://example.com/ep-1",
    ...partial,
  }
}

describe("heightLabel", () => {
  it("labels 2160 as 4K and others as Np", () => {
    expect(heightLabel(2160)).toBe("4K")
    expect(heightLabel(1440)).toBe("1440p")
    expect(heightLabel(1080)).toBe("1080p")
  })
})

describe("buildQualityOptions", () => {
  it("lists Original plus each variant descending, without a duplicate Upscale item", () => {
    const options = buildQualityOptions(
      episode({
        storageKey: "a.mp4",
        upscaledStorageKey: "a_up.mp4",
        upscaledVariants: [
          { height: 1080, storageKey: "a_up_1080p.mp4" },
          { height: 2160, storageKey: "a_up.mp4" },
          { height: 1440, storageKey: "a_up_1440p.mp4" },
        ],
      }),
    )
    expect(options).toEqual([
      { label: "Original", variant: "original" },
      { label: "4K", variant: "2160p" },
      { label: "1440p", variant: "1440p" },
      { label: "1080p", variant: "1080p" },
    ])
  })

  it("falls back to a single Upscale item when variants are missing", () => {
    const options = buildQualityOptions(episode({ storageKey: "a.mp4", upscaledStorageKey: "a_up.mp4" }))
    expect(options).toEqual([
      { label: "Original", variant: "original" },
      { label: "Upscale", variant: "upscaled" },
    ])
  })

  it("returns only Original when there is no upscale", () => {
    expect(buildQualityOptions(episode({ storageKey: "a.mp4" }))).toEqual([
      { label: "Original", variant: "original" },
    ])
  })
})
