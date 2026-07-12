import { describe, it, expect } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import App from "../../src/App"

describe("Anime Flow", () => {
  it("shows anime info after submitting a valid URL", async () => {
    const user = userEvent.setup()
    render(<App />)

    const input = screen.getByPlaceholderText(/animesonlinecc/i)
    await user.type(input, "https://animesonlinecc.to/anime/shingeki-no-kyojin/")
    await user.click(screen.getByRole("button", { name: /buscar/i }))

    await waitFor(() => {
      expect(screen.getByText("Shingeki no Kyojin")).toBeInTheDocument()
    }, { timeout: 3000 })
  })

  it("shows error message for invalid URL", async () => {
    const user = userEvent.setup()
    render(<App />)

    const input = screen.getByPlaceholderText(/animesonlinecc/i)
    await user.type(input, "https://animesonlinecc.to/anime/nonexistent/")
    await user.click(screen.getByRole("button", { name: /buscar/i }))

    await waitFor(() => {
      expect(screen.getByText(/não foi possível encontrar/i)).toBeInTheDocument()
    }, { timeout: 3000 })
  })

  it("displays episode tabs after loading anime", async () => {
    const user = userEvent.setup()
    render(<App />)

    const input = screen.getByPlaceholderText(/animesonlinecc/i)
    await user.type(input, "https://animesonlinecc.to/anime/shingeki-no-kyojin/")
    await user.click(screen.getByRole("button", { name: /buscar/i }))

    await waitFor(() => {
      expect(screen.getByText("Shingeki no Kyojin")).toBeInTheDocument()
    }, { timeout: 3000 })

    expect(screen.getByRole("tab", { name: /Temporada 1/i })).toBeInTheDocument()
  })
})
