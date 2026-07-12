import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { UrlInput } from "../../src/components/anime/url-input"

describe("UrlInput", () => {
  it("submits a valid http url", async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(<UrlInput onSubmit={onSubmit} isLoading={false} />)

    await user.type(screen.getByRole("textbox", { name: /url do anime/i }), "https://exemplo.com/anime/x")
    await user.click(screen.getByRole("button", { name: /buscar/i }))

    expect(onSubmit).toHaveBeenCalledWith("https://exemplo.com/anime/x")
  })

  it("rejects text that is not a url and explains the problem", async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(<UrlInput onSubmit={onSubmit} isLoading={false} />)

    await user.type(screen.getByRole("textbox", { name: /url do anime/i }), "shingeki no kyojin")
    await user.click(screen.getByRole("button", { name: /buscar/i }))

    expect(onSubmit).not.toHaveBeenCalled()
    expect(screen.getByRole("alert")).toHaveTextContent(/não parece uma URL/i)
  })

  it("rejects non-http protocols", async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(<UrlInput onSubmit={onSubmit} isLoading={false} />)

    await user.type(screen.getByRole("textbox", { name: /url do anime/i }), "ftp://exemplo.com/x")
    await user.click(screen.getByRole("button", { name: /buscar/i }))

    expect(onSubmit).not.toHaveBeenCalled()
    expect(screen.getByRole("alert")).toHaveTextContent(/http/i)
  })

  it("clears the error when the user edits the input", async () => {
    const user = userEvent.setup()
    render(<UrlInput onSubmit={() => undefined} isLoading={false} />)

    await user.type(screen.getByRole("textbox", { name: /url do anime/i }), "abc")
    await user.click(screen.getByRole("button", { name: /buscar/i }))
    expect(screen.getByRole("alert")).toBeInTheDocument()

    await user.type(screen.getByRole("textbox", { name: /url do anime/i }), "d")
    expect(screen.queryByRole("alert")).not.toBeInTheDocument()
  })

  it("submits on Enter", async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(<UrlInput onSubmit={onSubmit} isLoading={false} />)

    await user.type(
      screen.getByRole("textbox", { name: /url do anime/i }),
      "https://exemplo.com/anime/y{Enter}",
    )

    expect(onSubmit).toHaveBeenCalledWith("https://exemplo.com/anime/y")
  })
})
