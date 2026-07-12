import { describe, expect, it } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import App from "../../src/App"

async function openCatalogAndAnime(user: ReturnType<typeof userEvent.setup>) {
  render(<App />)
  await user.click(screen.getByRole("button", { name: /catálogo/i }))
  await waitFor(() => {
    expect(screen.getByText("Meu Catálogo")).toBeInTheDocument()
  })
  await user.click(screen.getAllByRole("button", { name: /Abrir Shingeki no Kyojin/i })[0])
}

describe("Episode Navigation", () => {
  it("shows next episode button when playing the first episode", async () => {
    const user = userEvent.setup()
    await openCatalogAndAnime(user)

    await user.click(screen.getByRole("button", { name: /Assistir Episódio 01/i }))

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Próximo episódio/i })).toBeInTheDocument()
    })
  })

  it("does not show previous episode button on the first episode", async () => {
    const user = userEvent.setup()
    await openCatalogAndAnime(user)

    await user.click(screen.getByRole("button", { name: /Assistir Episódio 01/i }))

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Próximo episódio/i })).toBeInTheDocument()
    })
    expect(screen.queryByRole("button", { name: /Episódio anterior/i })).not.toBeInTheDocument()
  })

  it("shows both navigation buttons on a middle episode", async () => {
    const user = userEvent.setup()
    await openCatalogAndAnime(user)

    await user.click(screen.getByRole("button", { name: /Assistir Episódio 02/i }))

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Próximo episódio/i })).toBeInTheDocument()
      expect(screen.getByRole("button", { name: /Episódio anterior/i })).toBeInTheDocument()
    })
  })

  it("does not show next episode button on the last downloaded episode", async () => {
    const user = userEvent.setup()
    await openCatalogAndAnime(user)

    await user.click(screen.getByRole("button", { name: /Assistir Episódio 03/i }))

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Episódio anterior/i })).toBeInTheDocument()
    })
    expect(screen.queryByRole("button", { name: /Próximo episódio/i })).not.toBeInTheDocument()
  })

  it("highlights the currently playing episode in the list", async () => {
    const user = userEvent.setup()
    await openCatalogAndAnime(user)

    await user.click(screen.getByRole("button", { name: /Assistir Episódio 01/i }))

    await waitFor(() => {
      const label = screen.getByText("Episódio 01")
      const row = label.closest("[class*='bg-primary']")
      expect(row).toBeTruthy()
    })
  })
})
