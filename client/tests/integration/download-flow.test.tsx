import { describe, it, expect, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import App from "../../src/App"
import { useDownloads } from "../../src/hooks/use-downloads"

async function loadAnimeAndGetCheckboxes(user: ReturnType<typeof userEvent.setup>) {
  const input = screen.getByPlaceholderText(/animesonlinecc/i)
  await user.type(input, "https://animesonlinecc.to/anime/one-punch-man/")
  await user.click(screen.getByRole("button", { name: /buscar/i }))
  await waitFor(() => {
    expect(screen.getByRole("heading", { name: "One Punch Man" })).toBeInTheDocument()
  }, { timeout: 3000 })
  return screen.getAllByRole("checkbox")
}

describe("Download Flow", () => {
  beforeEach(() => {
    useDownloads.setState({ downloads: {} })
  })

  it("enables download button after selecting episodes", async () => {
    const user = userEvent.setup()
    render(<App />)
    const checkboxes = await loadAnimeAndGetCheckboxes(user)

    const downloadBtn = screen.getByRole("button", { name: /baixar/i })
    expect(downloadBtn).toBeDisabled()

    await user.click(checkboxes[0])
    expect(downloadBtn).toBeEnabled()
  })

  it("shows download items after clicking download", async () => {
    const user = userEvent.setup()
    render(<App />)
    const checkboxes = await loadAnimeAndGetCheckboxes(user)

    await user.click(checkboxes[0])
    await user.click(checkboxes[1])
    await user.click(screen.getByRole("button", { name: /baixar/i }))

    await waitFor(() => {
      const downloads = Object.values(useDownloads.getState().downloads)
      expect(downloads.length).toBe(2)
    }, { timeout: 3000 })
  })

  it("defaults to creating a new anime with the scraped title", async () => {
    const user = userEvent.setup()
    render(<App />)
    const checkboxes = await loadAnimeAndGetCheckboxes(user)

    const titleInput = screen.getByLabelText(/nome do novo anime/i)
    expect(titleInput).toHaveValue("One Punch Man")

    await user.click(checkboxes[0])
    await user.click(screen.getByRole("button", { name: /baixar/i }))

    await waitFor(() => {
      const downloads = Object.values(useDownloads.getState().downloads)
      expect(downloads.length).toBe(1)
      expect(downloads[0].animeTitle).toBe("One Punch Man")
    }, { timeout: 3000 })
  })

  it("allocates downloads to an existing catalog anime", async () => {
    const user = userEvent.setup()
    render(<App />)
    const checkboxes = await loadAnimeAndGetCheckboxes(user)

    const select = await screen.findByLabelText(/anime de destino/i)
    await waitFor(() => {
      expect(screen.getByRole("option", { name: "Shingeki no Kyojin" })).toBeInTheDocument()
    }, { timeout: 3000 })
    await user.selectOptions(select, "shingeki-no-kyojin")

    expect(screen.queryByLabelText(/nome do novo anime/i)).not.toBeInTheDocument()

    await user.click(checkboxes[0])
    await user.click(screen.getByRole("button", { name: /baixar/i }))

    await waitFor(() => {
      const downloads = Object.values(useDownloads.getState().downloads)
      expect(downloads.length).toBe(1)
      expect(downloads[0].animeId).toBe("shingeki-no-kyojin")
      expect(downloads[0].animeTitle).toBe("Shingeki no Kyojin")
    }, { timeout: 3000 })
  })
})
