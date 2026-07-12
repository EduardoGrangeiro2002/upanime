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
    expect(screen.getByText("One Punch Man")).toBeInTheDocument()
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
})
