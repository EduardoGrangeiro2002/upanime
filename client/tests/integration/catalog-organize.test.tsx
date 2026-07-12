import { describe, expect, it } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import App from "../../src/App"

async function openAnimeDetail(user: ReturnType<typeof userEvent.setup>) {
  render(<App />)
  await user.click(screen.getByRole("button", { name: /catálogo/i }))
  await waitFor(() => {
    expect(screen.getByText("Meu Catálogo")).toBeInTheDocument()
  })
  await user.click(screen.getAllByRole("button", { name: /Abrir Shingeki no Kyojin/i })[0])
  await screen.findByRole("dialog")
}

describe("Catalog organize episodes", () => {
  it("organizes episodes via AI from the detail dialog", async () => {
    const user = userEvent.setup()
    await openAnimeDetail(user)

    await user.click(screen.getByRole("button", { name: /Organizar episódios com IA/i }))

    await waitFor(() => {
      expect(screen.getByText("Episódios organizados")).toBeInTheDocument()
    })
  })
})
