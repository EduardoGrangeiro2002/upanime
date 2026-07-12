import { describe, it, expect } from "vitest"
import { detectEpisodeNumber } from "../../src/lib/episode-number"

describe("detectEpisodeNumber", () => {
  it("uses the last number in the filename", () => {
    expect(detectEpisodeNumber("Black Lagoon - 03.mp4", 9)).toBe("3")
    expect(detectEpisodeNumber("S01E12.mkv", 9)).toBe("12")
    expect(detectEpisodeNumber("anime_ep_007_final.webm", 9)).toBe("7")
  })

  it("ignores the extension digits", () => {
    expect(detectEpisodeNumber("episodio-5.mp4", 9)).toBe("5")
  })

  it("falls back to the sequential index when there is no number", () => {
    expect(detectEpisodeNumber("abertura.mp4", 4)).toBe("4")
  })
})
