import { describe, expect, it } from "vitest"
import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import App from "../../src/App"

async function openUpscalePage(user: ReturnType<typeof userEvent.setup>) {
  render(<App />)
  await user.click(screen.getByRole("button", { name: "Upscale" }))
  await waitFor(() => {
    expect(screen.getByRole("heading", { level: 1, name: "Upscale" })).toBeInTheDocument()
  })
}

describe("Upscale Page", () => {
  it("shows the pipeline config with explained sliders", async () => {
    const user = userEvent.setup()
    await openUpscalePage(user)

    expect(screen.getAllByText(/Real-ESRGAN AnimeVideo v3/i).length).toBeGreaterThan(0)
    expect(screen.getByRole("slider", { name: "Batch" })).toBeInTheDocument()
    expect(screen.getByRole("slider", { name: "Nitidez" })).toBeInTheDocument()
    expect(screen.getByRole("slider", { name: "Saturação" })).toBeInTheDocument()
    expect(screen.getByRole("slider", { name: "Contraste" })).toBeInTheDocument()
  })

  it("shows the action bar only after selecting episodes and starts the job", async () => {
    const user = userEvent.setup()
    await openUpscalePage(user)

    expect(screen.queryByRole("button", { name: /Iniciar Upscale/i })).not.toBeInTheDocument()

    await user.click(await screen.findByRole("button", { name: /shingeki no kyojin/i }))
    await user.click(await screen.findByRole("button", { name: /selecionar todos/i }))

    const startButton = await screen.findByRole("button", { name: /Iniciar Upscale/i })
    await user.click(startButton)

    await waitFor(() => {
      const queueHeading = screen.getByRole("heading", { name: /Fila de Processamento/i })
      const queueSection = queueHeading.closest("div")!.parentElement!
      expect(within(queueSection).getAllByText(/Na fila|Processando/i).length).toBeGreaterThan(0)
    })

    expect(screen.queryByRole("button", { name: /Iniciar Upscale/i })).not.toBeInTheDocument()
  })
})
