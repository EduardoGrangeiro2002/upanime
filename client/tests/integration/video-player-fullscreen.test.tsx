import { describe, expect, it, afterEach, vi } from "vitest"
import { render, screen, act } from "@testing-library/react"
import { MediaPlayerInstance } from "@vidstack/react"
import { VideoPlayer } from "../../src/components/catalog/video-player"
import type { Episode } from "../../src/api/types"

function setFullscreenEnabled(value: boolean) {
  Object.defineProperty(document, "fullscreenEnabled", { value, configurable: true })
}

const episode: Episode = {
  id: "ep-1",
  title: "Ep 1",
  number: "1",
  seasonNumber: 1,
  type: "episode",
  url: "https://example.com/ep-1",
  storageKey: "a.mp4",
}

function renderPlayer() {
  return render(
    <VideoPlayer
      src="/video.mp4"
      title="Ep 1"
      episode={episode}
      resolveVariantUrl={(v) => `/video.mp4?variant=${v}`}
      onClose={() => {}}
    />,
  )
}

afterEach(() => {
  setFullscreenEnabled(false)
  vi.restoreAllMocks()
})

function pseudoLayer() {
  return [...document.body.querySelectorAll("div[style]")].find((el) =>
    (el.getAttribute("style") || "").includes("z-index: 9999"),
  )
}

describe("VideoPlayer real fullscreen (desktop)", () => {
  it("requests element fullscreen on the player when the api is available", async () => {
    setFullscreenEnabled(true)
    const enter = vi.spyOn(MediaPlayerInstance.prototype, "enterFullscreen").mockResolvedValue(undefined)
    renderPlayer()

    const button = await screen.findByRole("button", { name: "Tela cheia" })
    act(() => button.click())

    expect(enter).toHaveBeenCalledOnce()
    expect(pseudoLayer()).toBeUndefined()
  })

  it("exits element fullscreen when already fullscreen", async () => {
    setFullscreenEnabled(true)
    const enter = vi.spyOn(MediaPlayerInstance.prototype, "enterFullscreen").mockRejectedValue(new Error("denied"))
    renderPlayer()

    const button = await screen.findByRole("button", { name: "Tela cheia" })
    act(() => button.click())

    expect(enter).toHaveBeenCalledOnce()
  })
})

describe("VideoPlayer pseudo-fullscreen (iPhone-like)", () => {
  it("expands into a full-window backdrop layer instead of calling the fullscreen api", async () => {
    setFullscreenEnabled(false)
    const enter = vi.spyOn(MediaPlayerInstance.prototype, "enterFullscreen").mockResolvedValue(undefined)
    renderPlayer()

    const button = await screen.findByRole("button", { name: "Tela cheia" })
    act(() => {
      button.click()
    })

    expect(enter).not.toHaveBeenCalled()
    expect(screen.getByRole("button", { name: "Sair da tela cheia" })).toBeInTheDocument()
    expect(pseudoLayer()).toBeDefined()
  })

  it("paints the browser chrome black while maximized and restores it on exit", async () => {
    setFullscreenEnabled(false)
    renderPlayer()

    const button = await screen.findByRole("button", { name: "Tela cheia" })
    act(() => button.click())

    expect(document.querySelector('meta[name="theme-color"]')?.getAttribute("content")).toBe("#000000")

    const exit = screen.getByRole("button", { name: "Sair da tela cheia" })
    act(() => exit.click())

    expect(document.querySelector('meta[name="theme-color"]')?.getAttribute("content")).not.toBe("#000000")
  })

  it("toggles back out of pseudo-fullscreen on a second tap", async () => {
    setFullscreenEnabled(false)
    renderPlayer()

    const button = await screen.findByRole("button", { name: "Tela cheia" })
    act(() => button.click())
    const exit = screen.getByRole("button", { name: "Sair da tela cheia" })
    act(() => exit.click())

    expect(screen.getByRole("button", { name: "Tela cheia" })).toBeInTheDocument()
    expect(pseudoLayer()).toBeUndefined()
  })

  it("keeps the same video element (no remount) when toggling fullscreen", async () => {
    setFullscreenEnabled(false)
    renderPlayer()

    const button = await screen.findByRole("button", { name: "Tela cheia" })
    const playerBefore = document.body.querySelector("[data-media-player]")
    act(() => button.click())
    const playerAfter = document.body.querySelector("[data-media-player]")

    expect(playerAfter).toBe(playerBefore)
  })
})

describe("VideoPlayer sliders", () => {
  it("wires the vidstack --slider-fill var into fills and thumbs", () => {
    renderPlayer()
    const wired = document.body.querySelectorAll("[style*='var(--slider-fill)']")
    expect(wired.length).toBe(4)
  })
})
