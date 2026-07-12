import { describe, expect, it } from "vitest"
import { isAggressiveUpscale, maxSafeTargetHeight } from "../../src/lib/upscale-warning"

describe("isAggressiveUpscale", () => {
  it("flags VHS-range sources against HD targets", () => {
    expect(isAggressiveUpscale(240, 1080)).toBe(true)
    expect(isAggressiveUpscale(330, 2160)).toBe(true)
  })

  it("accepts ratios up to 3x", () => {
    expect(isAggressiveUpscale(360, 1080)).toBe(false)
    expect(isAggressiveUpscale(480, 1440)).toBe(false)
    expect(isAggressiveUpscale(1080, 2160)).toBe(false)
  })

  it("flags DVD sources only against 4K", () => {
    expect(isAggressiveUpscale(480, 1080)).toBe(false)
    expect(isAggressiveUpscale(480, 2160)).toBe(true)
  })

  it("ignores unknown or invalid source heights", () => {
    expect(isAggressiveUpscale(0, 2160)).toBe(false)
    expect(isAggressiveUpscale(-1, 2160)).toBe(false)
  })
})

describe("maxSafeTargetHeight", () => {
  it("suggests the largest non-aggressive target", () => {
    expect(maxSafeTargetHeight(480, [1080, 1440, 2160])).toBe(1440)
    expect(maxSafeTargetHeight(720, [1080, 1440, 2160])).toBe(2160)
  })

  it("returns null when every target is aggressive", () => {
    expect(maxSafeTargetHeight(240, [1080, 1440, 2160])).toBe(null)
  })
})
