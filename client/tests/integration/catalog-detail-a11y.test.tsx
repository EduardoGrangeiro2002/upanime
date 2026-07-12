import { describe, expect, it } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import App from "../../src/App"

async function openCatalog(user: ReturnType<typeof userEvent.setup>) {
  render(<App />)
  await user.click(screen.getByRole("button", { name: /catálogo/i }))
  await waitFor(() => {
    expect(screen.getByText("Meu Catálogo")).toBeInTheDocument()
  })
}

describe("Catalog Detail dialog", () => {
  it("opens as a real dialog and locks body scroll", async () => {
    const user = userEvent.setup()
    await openCatalog(user)

    await user.click(screen.getAllByRole("button", { name: /Abrir Shingeki no Kyojin/i })[0])

    const dialog = await screen.findByRole("dialog")
    expect(dialog).toHaveAttribute("aria-modal", "true")
    expect(dialog).toHaveAccessibleName(/Shingeki no Kyojin/i)
    expect(document.body.style.overflow).toBe("hidden")
  })

  it("closes on Escape and restores scroll", async () => {
    const user = userEvent.setup()
    await openCatalog(user)

    await user.click(screen.getAllByRole("button", { name: /Abrir Shingeki no Kyojin/i })[0])
    await screen.findByRole("dialog")

    await user.keyboard("{Escape}")

    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    })
    expect(document.body.style.overflow).toBe("")
  })

  it("moves focus into the dialog when it opens", async () => {
    const user = userEvent.setup()
    await openCatalog(user)

    await user.click(screen.getAllByRole("button", { name: /Abrir Shingeki no Kyojin/i })[0])
    const dialog = await screen.findByRole("dialog")

    await waitFor(() => {
      expect(dialog.contains(document.activeElement)).toBe(true)
    })
  })

  it("updates the URL hash for deep-linking and restores it on close", async () => {
    const user = userEvent.setup()
    await openCatalog(user)

    await user.click(screen.getAllByRole("button", { name: /Abrir Shingeki no Kyojin/i })[0])
    await screen.findByRole("dialog")

    expect(window.location.hash).toMatch(/^#\/catalogo\/.+/)

    await user.keyboard("{Escape}")
    await waitFor(() => {
      expect(window.location.hash).toBe("#/catalogo")
    })
  })

  it("opens the dialog directly from a deep link", async () => {
    window.location.hash = "#/catalogo/shingeki-no-kyojin"
    render(<App />)

    const dialog = await screen.findByRole("dialog")
    expect(dialog).toHaveAccessibleName(/Shingeki no Kyojin/i)
  })
})
