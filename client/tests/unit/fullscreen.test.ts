import { describe, expect, it, afterEach } from "vitest"
import { canUseRealFullscreen, needsCssRotation, pseudoFullscreenStyle } from "../../src/lib/fullscreen"

function setFullscreenEnabled(value: boolean) {
  Object.defineProperty(document, "fullscreenEnabled", { value, configurable: true })
}

function setOrientation(portrait: boolean) {
  window.matchMedia = ((query: string) => ({
    matches: query.includes("portrait") ? portrait : !portrait,
    media: query,
    onchange: null,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    addListener: () => undefined,
    removeListener: () => undefined,
    dispatchEvent: () => false,
  })) as typeof window.matchMedia
}

afterEach(() => {
  setFullscreenEnabled(false)
})

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

describe("needsCssRotation", () => {
  it("is true in portrait", () => {
    setOrientation(true)
    expect(needsCssRotation()).toBe(true)
  })

  it("is false in landscape", () => {
    setOrientation(false)
    expect(needsCssRotation()).toBe(false)
  })
})

describe("pseudoFullscreenStyle", () => {
  it("fills the viewport without rotation", () => {
    const style = pseudoFullscreenStyle(false)
    expect(style.position).toBe("fixed")
    expect(style.width).toBe("100dvw")
    expect(style.height).toBe("100dvh")
    expect(style.transform).toBeUndefined()
    expect(style.zIndex).toBe(9999)
  })

  it("swaps dimensions and rotates 90deg with the exact transform", () => {
    const style = pseudoFullscreenStyle(true)
    expect(style.width).toBe("100dvh")
    expect(style.height).toBe("100dvw")
    expect(style.transform).toBe("translate(100dvw, 0) rotate(90deg)")
    expect(style.transformOrigin).toBe("top left")
  })
})
