import { describe, it, expect } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import App from "../../src/App"

async function openUploadTab(user: ReturnType<typeof userEvent.setup>) {
  render(<App />)
  await user.click(screen.getByRole("tab", { name: /enviar arquivos/i }))
}

describe("Upload flow", () => {
  it("uploads selected files and marks them as done", async () => {
    const user = userEvent.setup()
    await openUploadTab(user)

    await user.type(screen.getByRole("textbox", { name: /nome do anime/i }), "Meu Anime Local")

    const file = new File(["video"], "meu-anime-01.mp4", { type: "video/mp4" })
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    await user.upload(input, file)

    expect(screen.getByText("meu-anime-01.mp4")).toBeInTheDocument()
    expect(screen.getByText("Ep 1")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: /enviar 1 episódio/i }))

    await waitFor(() => {
      expect(screen.getByText(/1 episódio enviado para o catálogo/i)).toBeInTheDocument()
    })
  })

  it("detects episode numbers from filenames", async () => {
    const user = userEvent.setup()
    await openUploadTab(user)

    const files = [
      new File(["a"], "anime-ep-03.mp4", { type: "video/mp4" }),
      new File(["b"], "anime-ep-04.mp4", { type: "video/mp4" }),
    ]
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    await user.upload(input, files)

    expect(screen.getByText("Ep 3")).toBeInTheDocument()
    expect(screen.getByText("Ep 4")).toBeInTheDocument()
  })

  it("keeps the submit button disabled without a title", async () => {
    const user = userEvent.setup()
    await openUploadTab(user)

    const file = new File(["video"], "ep1.mp4", { type: "video/mp4" })
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    await user.upload(input, file)

    expect(screen.getByRole("button", { name: /enviar 1 episódio/i })).toBeDisabled()
  })

  it("uploaded anime shows up in the catalog", async () => {
    const user = userEvent.setup()
    await openUploadTab(user)

    await user.type(screen.getByRole("textbox", { name: /nome do anime/i }), "Anime Enviado")
    const file = new File(["video"], "ep-01.mp4", { type: "video/mp4" })
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    await user.upload(input, file)
    await user.click(screen.getByRole("button", { name: /enviar 1 episódio/i }))

    await waitFor(() => {
      expect(screen.getByText(/1 episódio enviado para o catálogo/i)).toBeInTheDocument()
    })

    await user.click(screen.getByRole("button", { name: /catálogo/i }))
    await waitFor(() => {
      expect(screen.getByText("Meu Catálogo")).toBeInTheDocument()
    })
    const cards = await screen.findAllByRole("button", { name: /Abrir Anime Enviado/i })
    expect(cards.length).toBeGreaterThan(0)
    expect(screen.getByRole("heading", { name: "Sem categoria" })).toBeInTheDocument()
  })
})
