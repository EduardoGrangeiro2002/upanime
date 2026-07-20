import { describe, expect, it } from "vitest"
import { render, screen, act } from "@testing-library/react"
import { VideoPlayer } from "../../src/components/catalog/video-player"
import type { Episode } from "../../src/api/types"

const episode: Episode = {
  id: "ep-1",
  title: "Ep 1",
  number: "1",
  seasonNumber: 1,
  type: "episode",
  url: "https://example.com/ep-1",
  storageKey: "a.mp4",
  upscaledStorageKey: "a_up.mp4",
  upscaledVariants: [
    { height: 2160, storageKey: "a_up.mp4" },
    { height: 1080, storageKey: "a_up_1080p.mp4" },
  ],
}

function findVideoSrc() {
  const provider = document.querySelector("[data-media-provider]")
  const player = document.querySelector("[data-media-player]")
  return { provider, player }
}

describe("Quality switch", () => {
  it("lists the real resolutions and switches source without unmounting the player", async () => {
    const requested: string[] = []
    render(
      <VideoPlayer
        src="/api/catalog/episode/ep-1/stream/file?variant=original"
        title="Ep 1"
        episode={episode}
        resolveVariantUrl={(v) => {
          const url = `/api/catalog/episode/ep-1/stream/file?variant=${v}`
          requested.push(url)
          return url
        }}
        onClose={() => {}}
      />,
    )

    const { player: before } = findVideoSrc()
    const gear = await screen.findByRole("button", { name: "Qualidade" })
    act(() => gear.click())

    const menu = screen.getByRole("menu")
    expect(menu.querySelectorAll("[role=menuitemradio]")).toHaveLength(3)
    const option1080 = screen.getByRole("menuitemradio", { name: /1080p/ })

    act(() => (option1080 as HTMLElement).click())

    expect(requested).toContain("/api/catalog/episode/ep-1/stream/file?variant=1080p")
    const { player: after } = findVideoSrc()
    expect(after).toBe(before)
  })
})
