import { describe, it, expect } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import App from "../../src/App"

async function loadAnime(user: ReturnType<typeof userEvent.setup>, url: string) {
  const input = screen.getByPlaceholderText(/animesonlinecc/i)
  await user.type(input, url)
  await user.click(screen.getByRole("button", { name: /buscar/i }))
  await waitFor(() => {
    expect(screen.getByRole("heading", { level: 3, name: /episódios/i })).toBeInTheDocument()
  }, { timeout: 3000 })
}

describe("Episode Tabs", () => {
  it("switches between season tabs", async () => {
    const user = userEvent.setup()
    render(<App />)
    await loadAnime(user, "https://animesonlinecc.to/anime/shingeki-no-kyojin/")

    const tab2 = screen.getByRole("tab", { name: /Temporada 2/i })
    await user.click(tab2)

    expect(tab2).toHaveAttribute("aria-selected", "true")
  })

  it("shows OVA tab for anime with OVAs", async () => {
    const user = userEvent.setup()
    render(<App />)
    await loadAnime(user, "https://animesonlinecc.to/anime/shingeki-no-kyojin/")

    expect(screen.getByRole("tab", { name: /OVAs/i })).toBeInTheDocument()
  })

  it("shows Movies tab for anime with movies", async () => {
    const user = userEvent.setup()
    render(<App />)
    await loadAnime(user, "https://animesonlinecc.to/anime/one-punch-man/")

    expect(screen.getByRole("tab", { name: /Filmes/i })).toBeInTheDocument()
  })
})
