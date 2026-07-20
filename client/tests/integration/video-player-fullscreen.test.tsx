import { describe, expect, it, afterEach } from "vitest"
import { render, screen, act } from "@testing-library/react"
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
})

describe("VideoPlayer pseudo-fullscreen (iPhone-like)", () => {
  it("portals the player into a fixed full-window layer when maximizing without the fullscreen api", async () => {
    setFullscreenEnabled(false)
    renderPlayer()

    const button = await screen.findByRole("button", { name: "Tela cheia" })
    act(() => {
      button.click()
    })

    const exit = screen.getByRole("button", { name: "Sair da tela cheia" })
    const layer = exit.closest("[style]") as HTMLElement | null
    const fixedLayer = document.body.querySelector<HTMLElement>(":scope > div[style*='fixed']")
    expect(fixedLayer).not.toBeNull()
    expect(layer).not.toBeNull()
  })

  it("toggles back out of pseudo-fullscreen on a second tap", async () => {
    setFullscreenEnabled(false)
    renderPlayer()

    const button = await screen.findByRole("button", { name: "Tela cheia" })
    act(() => button.click())
    const exit = screen.getByRole("button", { name: "Sair da tela cheia" })
    act(() => exit.click())

    expect(screen.getByRole("button", { name: "Tela cheia" })).toBeInTheDocument()
    expect(document.body.querySelector(":scope > div[style*='fixed']")).toBeNull()
  })
})
