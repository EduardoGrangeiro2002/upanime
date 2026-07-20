import { describe, expect, it } from "vitest"
import { canUseRealFullscreen, pseudoBackdropStyle, pseudoFullscreenStyle } from "../../src/lib/fullscreen"

function setFullscreenEnabled(value: boolean) {
  Object.defineProperty(document, "fullscreenEnabled", { value, configurable: true })
}

describe("canUseRealFullscreen", () => {
  it("is true when the document supports the fullscreen api", () => {
    setFullscreenEnabled(true)
    expect(canUseRealFullscreen()).toBe(true)
  })

  it("is false when the document does not (iPhone)", () => {
    setFullscreenEnabled(false)
    expect(canUseRealFullscreen()).toBe(false)
  })
})

describe("pseudoBackdropStyle", () => {
  it("covers the whole viewport in black behind the player", () => {
    const style = pseudoBackdropStyle()
    expect(style.position).toBe("fixed")
    expect(style.inset).toBe(0)
    expect(style.background).toBe("#000")
  })
})

describe("pseudoFullscreenStyle", () => {
  it("fills the viewport as-is in landscape", () => {
    const style = pseudoFullscreenStyle({ width: 844, height: 390 })
    expect(style.width).toBe(844)
    expect(style.height).toBe(390)
    expect(style.transform).toBeUndefined()
  })

  it("swaps dimensions and rotates with exact pixel translate in portrait", () => {
    const style = pseudoFullscreenStyle({ width: 390, height: 844 })
    expect(style.width).toBe(844)
    expect(style.height).toBe(390)
    expect(style.transform).toBe("translate(390px, 0px) rotate(90deg)")
    expect(style.transformOrigin).toBe("top left")
  })
})
