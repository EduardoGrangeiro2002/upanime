import { describe, expect, it, afterEach } from "vitest"
import { render, act } from "@testing-library/react"
import { VideoPlayer, needsRotatedFullscreen } from "../../src/components/catalog/video-player"

function setFullscreenEnabled(value: boolean | undefined) {
  Object.defineProperty(document, "fullscreenEnabled", { value, configurable: true })
}

function setOrientationLock(lock: (() => Promise<void>) | undefined) {
  Object.defineProperty(screen, "orientation", {
    value: lock ? { lock } : undefined,
    configurable: true,
  })
}

function renderPlayer() {
  return render(
    <VideoPlayer src="/video.mp4" title="Ep 1" episodeId="ep-1" onClose={() => {}} />,
  )
}

function dispatchFullscreenRequest(container: HTMLElement) {
  const player = container.querySelector("[data-media-player]")
  expect(player).not.toBeNull()
  const event = new Event("media-enter-fullscreen-request", { bubbles: true, cancelable: true })
  act(() => {
    player!.dispatchEvent(event)
  })
  return event
}

afterEach(() => {
  setFullscreenEnabled(undefined)
  setOrientationLock(undefined)
})

describe("needsRotatedFullscreen", () => {
  it("is false when the fullscreen api is available", () => {
    setFullscreenEnabled(true)
    expect(needsRotatedFullscreen()).toBe(false)
  })

  it("is false when orientation lock is available", () => {
    setFullscreenEnabled(false)
    setOrientationLock(async () => {})
    expect(needsRotatedFullscreen()).toBe(false)
  })

  it("is true when neither fullscreen nor orientation lock exist", () => {
    setFullscreenEnabled(false)
    setOrientationLock(undefined)
    expect(needsRotatedFullscreen()).toBe(true)
  })
})

describe("VideoPlayer rotated fullscreen", () => {
  it("intercepts the fullscreen request and rotates to landscape on iPhone-like devices", () => {
    setFullscreenEnabled(false)
    setOrientationLock(undefined)
    const { container } = renderPlayer()

    const event = dispatchFullscreenRequest(container)

    expect(event.defaultPrevented).toBe(true)
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.className).toContain("fixed")
    expect(container.querySelector(".rotate-90")).not.toBeNull()
  })

  it("toggles back to inline on a second request", () => {
    setFullscreenEnabled(false)
    setOrientationLock(undefined)
    const { container } = renderPlayer()

    dispatchFullscreenRequest(container)
    dispatchFullscreenRequest(container)

    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.className).toContain("relative")
    expect(container.querySelector(".rotate-90")).toBeNull()
  })

  it("does not intercept when real fullscreen is available", () => {
    setFullscreenEnabled(true)
    const { container } = renderPlayer()

    const event = dispatchFullscreenRequest(container)

    expect(event.defaultPrevented).toBe(false)
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.className).toContain("relative")
  })
})
