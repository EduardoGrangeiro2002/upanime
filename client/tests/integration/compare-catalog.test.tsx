import { describe, it, expect } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import App from "../../src/App"

async function openComparePage(user: ReturnType<typeof userEvent.setup>) {
  render(<App />)
  await user.click(screen.getByRole("button", { name: /comparar/i }))
  await waitFor(() => {
    expect(screen.getByRole("heading", { name: /comparar vídeos/i })).toBeInTheDocument()
  })
}

describe("Compare from catalog", () => {
  it("loads original and upscaled streams after picking an episode", async () => {
    const user = userEvent.setup()
    await openComparePage(user)

    const animeSelect = await screen.findByRole("combobox", { name: /anime/i })
    await user.selectOptions(animeSelect, "shingeki-no-kyojin")

    const episodeSelect = screen.getByRole("combobox", { name: /episódio/i })
    await user.selectOptions(episodeSelect, "shingeki-no-kyojin-s1-e1")

    await waitFor(() => {
      const videos = document.querySelectorAll("video")
      expect(videos).toHaveLength(2)
      expect(videos[0].src).toContain("variant=original")
      expect(videos[1].src).toContain("variant=upscaled")
    })

    expect(screen.getByRole("button", { name: /play/i })).toBeInTheDocument()
  })

  it("only lists animes that have upscaled episodes", async () => {
    const user = userEvent.setup()
    await openComparePage(user)

    const animeSelect = await screen.findByRole("combobox", { name: /anime/i })
    const options = Array.from(animeSelect.querySelectorAll("option")).map((o) => o.textContent)

    expect(options).toContain("Shingeki no Kyojin")
    expect(options).not.toContain("One Punch Man")
  })

  it("switches to manual upload mode", async () => {
    const user = userEvent.setup()
    await openComparePage(user)

    await user.click(screen.getByRole("button", { name: /upload manual/i }))

    expect(screen.getByText(/Vídeo A \(Original\)/i)).toBeInTheDocument()
    expect(screen.getByText(/Vídeo B \(Processado\)/i)).toBeInTheDocument()
  })
})

describe("Catalog genre rows", () => {
  it("groups animes into genre rows", async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.click(screen.getByRole("button", { name: /catálogo/i }))

    await waitFor(() => {
      expect(screen.getByText("Meu Catálogo")).toBeInTheDocument()
    })

    expect(screen.getByRole("heading", { name: "Ação" })).toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "Drama" })).toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "Comédia" })).toBeInTheDocument()
  })
})
