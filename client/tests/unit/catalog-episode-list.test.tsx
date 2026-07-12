import { describe, expect, it, vi } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "../helpers"
import { CatalogEpisodeList } from "../../src/components/catalog/catalog-episode-list"
import type { Season } from "../../src/api/types"

vi.mock("@/hooks/use-catalog", () => ({
  useEpisodeStream: () => ({ data: null }),
}))

function buildSeason(count: number): Season {
  return {
    number: 1,
    label: "Temporada 1",
    type: "episode",
    episodes: Array.from({ length: count }, (_, i) => ({
      id: `ep-${i + 1}`,
      title: `Episódio ${i + 1}`,
      number: String(i + 1),
      seasonNumber: 1,
      type: "episode" as const,
      url: `https://example.com/ep-${i + 1}`,
      storageKey: `animes/test/ep-${i + 1}.mp4`,
    })),
  }
}

const defaultProps = {
  animeTitle: "Teste",
  onDeleteEpisode: () => undefined,
  onDeleteUpscaledEpisode: () => undefined,
  isDeleting: false,
  isDeletingUpscaled: false,
}

describe("CatalogEpisodeList", () => {
  it("shows the delete upscaled action when the episode has an upscaled asset", () => {
    const season: Season = {
      number: 1,
      label: "Temporada 1",
      type: "episode",
      episodes: [
        {
          id: "ep-1",
          title: "Episódio 1",
          number: "1",
          seasonNumber: 1,
          type: "episode",
          url: "https://example.com/ep-1",
          storageKey: "animes/test/ep-1.mp4",
          upscaledStorageKey: "animes/test/ep-1_upscaled.mp4",
        },
      ],
    }

    renderWithProviders(
      <CatalogEpisodeList
        {...defaultProps}
        season={season}
      />,
    )

    expect(screen.getByRole("button", { name: /Excluir upscale/i })).toBeInTheDocument()
  })

  it("renders all downloaded episodes", () => {
    const season = buildSeason(5)

    renderWithProviders(
      <CatalogEpisodeList
        {...defaultProps}
        season={season}
      />,
    )

    for (let i = 1; i <= 5; i++) {
      expect(screen.getByText(`Episódio 0${i}`)).toBeInTheDocument()
    }
  })

  it("renders the scrollable episode list container", () => {
    const season = buildSeason(3)

    const { container } = renderWithProviders(
      <CatalogEpisodeList
        {...defaultProps}
        season={season}
      />,
    )

    const scrollable = container.querySelector(".max-h-\\[240px\\]")
    expect(scrollable).toBeInTheDocument()
  })

  it("renders a lazy thumbnail for every episode row", () => {
    const season = buildSeason(3)

    const { container } = renderWithProviders(
      <CatalogEpisodeList
        {...defaultProps}
        season={season}
      />,
    )

    const thumbs = container.querySelectorAll("img[src^='/api/catalog/episode/']")
    expect(thumbs).toHaveLength(3)
    expect(thumbs[0]).toHaveAttribute("src", "/api/catalog/episode/ep-1/thumbnail")
    expect(thumbs[0]).toHaveAttribute("loading", "lazy")
  })

  it("hides the broken thumbnail and keeps the play button usable", () => {
    const season = buildSeason(1)

    const { container } = renderWithProviders(
      <CatalogEpisodeList
        {...defaultProps}
        season={season}
      />,
    )

    const img = container.querySelector("img[src='/api/catalog/episode/ep-1/thumbnail']") as HTMLImageElement
    img.dispatchEvent(new Event("error"))

    expect(img.style.display).toBe("none")
    expect(screen.getByRole("button", { name: /Assistir Episódio 01/i })).toBeInTheDocument()
  })
})
