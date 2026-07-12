import { describe, expect, it } from "vitest"
import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import App from "../../src/App"

async function openCatalog(user: ReturnType<typeof userEvent.setup>) {
  render(<App />)
  await user.click(screen.getByRole("button", { name: /catálogo/i }))
  await waitFor(() => {
    expect(screen.getByText("Meu Catálogo")).toBeInTheDocument()
  })
}

describe("Catalog Upscaled Flow", () => {
  it("shows original and upscaled controls for episodes with upscaled assets", async () => {
    const user = userEvent.setup()
    await openCatalog(user)

    await user.click(screen.getAllByRole("button", { name: /Abrir Shingeki no Kyojin/i })[0])
    await user.click(screen.getByRole("button", { name: /Assistir Episódio 01/i }))

    await waitFor(() => {
      const dialog = screen.getByRole("dialog")
      expect(within(dialog).getByRole("button", { name: "Original" })).toBeInTheDocument()
      expect(within(dialog).getByRole("button", { name: "Upscale" })).toBeInTheDocument()
    })
  })

  it("asks for confirmation before deleting the upscaled asset", async () => {
    const user = userEvent.setup()
    await openCatalog(user)

    await user.click(screen.getAllByRole("button", { name: /Abrir Shingeki no Kyojin/i })[0])
    await user.click(screen.getByRole("button", { name: /Excluir upscale/i }))

    expect(screen.getByRole("button", { name: /Remover upscale/i })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "Não" }))
    expect(screen.queryByRole("button", { name: /Remover upscale/i })).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Excluir upscale/i })).toBeInTheDocument()
  })

  it("removes only the upscaled option after deleting the upscaled asset", async () => {
    const user = userEvent.setup()
    await openCatalog(user)

    await user.click(screen.getAllByRole("button", { name: /Abrir Shingeki no Kyojin/i })[0])
    await user.click(screen.getByRole("button", { name: /Excluir upscale/i }))
    await user.click(screen.getByRole("button", { name: /Remover upscale/i }))
    await waitFor(() => {
      expect(screen.queryByRole("button", { name: /Excluir upscale/i })).not.toBeInTheDocument()
    })
    await user.click(screen.getByRole("button", { name: /Assistir Episódio 01/i }))

    const dialog = screen.getByRole("dialog")
    await within(dialog).findByRole("button", { name: /Fechar player/i })
    expect(within(dialog).queryByRole("button", { name: "Upscale" })).not.toBeInTheDocument()
  })
})
