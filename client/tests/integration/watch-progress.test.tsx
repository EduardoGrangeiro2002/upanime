import type { ReactNode } from "react"
import { describe, expect, it } from "vitest"
import { render, renderHook, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import App from "../../src/App"
import { renderWithProviders } from "../helpers"
import { CatalogEpisodeList } from "../../src/components/catalog/catalog-episode-list"
import { usePlaybackProgress } from "../../src/hooks/use-playback-progress"
import { seedWatchProgress, getMockWatchProgress } from "../../src/mocks/handlers"
import type { Season } from "../../src/api/types"

function shingekiSeason(): Season {
  return {
    number: 1,
    label: "Temporada 1",
    type: "episode",
    episodes: [1, 2, 3].map((n) => ({
      id: `shingeki-no-kyojin-s1-e${n}`,
      title: `Episódio ${n}`,
      number: String(n),
      seasonNumber: 1,
      type: "episode" as const,
      url: `https://animesonlinecc.to/episodio/shingeki-no-kyojin-1x${n}`,
      storageKey: `animes/shingeki-no-kyojin/shingeki-no-kyojin-s1-e${n}.mp4`,
    })),
  }
}

const listProps = {
  animeTitle: "Shingeki no Kyojin",
  onDeleteEpisode: () => undefined,
  onDeleteUpscaledEpisode: () => undefined,
  isDeleting: false,
  isDeletingUpscaled: false,
}

function hookWrapper({ children }: { children: ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
}

describe("Watch Progress", () => {
  it("shows where the user stopped based on server progress", async () => {
    seedWatchProgress("shingeki-no-kyojin-s1-e2", 120, 1400)
    const user = userEvent.setup()

    render(<App />)
    await user.click(screen.getByRole("button", { name: /catálogo/i }))
    await waitFor(() => {
      expect(screen.getByText("Meu Catálogo")).toBeInTheDocument()
    })
    await user.click(screen.getAllByRole("button", { name: /Abrir Shingeki no Kyojin/i })[0])

    expect(await screen.findByText("Parou em 2:00")).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByTestId("progress-shingeki-no-kyojin-s1-e2")).toHaveStyle({ width: "9%" })
    })
  })

  it("does not show a progress indicator for finished episodes", async () => {
    seedWatchProgress("shingeki-no-kyojin-s1-e2", 1390, 1400)
    const user = userEvent.setup()

    render(<App />)
    await user.click(screen.getByRole("button", { name: /catálogo/i }))
    await waitFor(() => {
      expect(screen.getByText("Meu Catálogo")).toBeInTheDocument()
    })
    await user.click(screen.getAllByRole("button", { name: /Abrir Shingeki no Kyojin/i })[0])

    await screen.findByRole("button", { name: /Assistir Episódio 01/i })
    expect(screen.queryByText(/Parou em/)).not.toBeInTheDocument()
  })

  it("auto-plays the in-progress episode when opening with autoplay", async () => {
    seedWatchProgress("shingeki-no-kyojin-s1-e2", 120, 1400)

    const { container } = renderWithProviders(
      <CatalogEpisodeList {...listProps} season={shingekiSeason()} autoPlayOnOpen />,
    )

    await waitFor(() => {
      const activeRow = container.querySelector(".bg-primary\\/10")
      expect(activeRow).not.toBeNull()
      expect(activeRow?.textContent).toContain("Episódio 02")
    })
  })

  it("resumes from the saved position and flushes progress on unmount", async () => {
    seedWatchProgress("shingeki-no-kyojin-s1-e1", 300, 1400)

    const { result, unmount } = renderHook(() => usePlaybackProgress("shingeki-no-kyojin-s1-e1"), {
      wrapper: hookWrapper,
    })

    await waitFor(() => {
      expect(result.current.ready).toBe(true)
      expect(result.current.savedTime).toBe(300)
    })

    result.current.handleTimeUpdate(310, 1400)
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(getMockWatchProgress("shingeki-no-kyojin-s1-e1")?.position).toBe(300)

    unmount()
    await waitFor(() => {
      expect(getMockWatchProgress("shingeki-no-kyojin-s1-e1")?.position).toBe(310)
    })
  })

  it("starts from zero when there is no saved progress", async () => {
    const { result } = renderHook(() => usePlaybackProgress("shingeki-no-kyojin-s1-e3"), {
      wrapper: hookWrapper,
    })

    await waitFor(() => {
      expect(result.current.ready).toBe(true)
    })
    expect(result.current.savedTime).toBe(0)
  })

  it("flushes progress when the page is hidden", async () => {
    const { result } = renderHook(() => usePlaybackProgress("shingeki-no-kyojin-s1-e1"), {
      wrapper: hookWrapper,
    })

    await waitFor(() => {
      expect(result.current.ready).toBe(true)
    })

    result.current.handleTimeUpdate(42, 1400)
    window.dispatchEvent(new Event("pagehide"))

    await waitFor(() => {
      expect(getMockWatchProgress("shingeki-no-kyojin-s1-e1")?.position).toBe(42)
    })
  })
})
