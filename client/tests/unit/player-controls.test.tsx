import { describe, expect, it, vi } from "vitest"
import { useState } from "react"
import { render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MediaPlayer, MediaProvider } from "@vidstack/react"
import { PlayerControls } from "../../src/components/catalog/player-controls"

function renderControls(overrides: Partial<Parameters<typeof PlayerControls>[0]> = {}) {
  function Harness() {
    const [menuOpen, setMenuOpen] = useState(overrides.menuOpen ?? false)
    return (
      <MediaPlayer src={{ src: "/video.mp4", type: "video/mp4" }} title="Ep">
        <MediaProvider />
        <PlayerControls
          onClose={() => {}}
          onPrevious={() => {}}
          onNext={() => {}}
          qualities={[
            { label: "Original", variant: "original" },
            { label: "1080p", variant: "1080p" },
          ]}
          activeQuality="original"
          onSelectQuality={() => {}}
          isFullscreen={false}
          onToggleFullscreen={() => {}}
          {...overrides}
          menuOpen={menuOpen}
          onMenuOpenChange={(open) => {
            overrides.onMenuOpenChange?.(open)
            setMenuOpen(open)
          }}
        />
      </MediaPlayer>
    )
  }
  return render(<Harness />)
}

describe("PlayerControls", () => {
  it("renders every pt-BR control label", () => {
    renderControls()
    for (const label of ["Fechar player", "Episódio anterior", "Próximo episódio", "Avançar 10 segundos", "Voltar 10 segundos", "Qualidade", "Tela cheia"]) {
      expect(screen.getByRole("button", { name: label })).toBeInTheDocument()
    }
  })

  it("shows the exit label when already in fullscreen", () => {
    renderControls({ isFullscreen: true })
    expect(screen.getByRole("button", { name: "Sair da tela cheia" })).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "Tela cheia" })).not.toBeInTheDocument()
  })

  it("hides the quality gear when there is a single option", () => {
    renderControls({ qualities: [{ label: "Original", variant: "original" }] })
    expect(screen.queryByRole("button", { name: "Qualidade" })).not.toBeInTheDocument()
  })

  it("wires the vidstack --slider-fill var into the seek and volume fills and thumbs", () => {
    const { container } = renderControls()
    const wired = [...container.querySelectorAll("[style]")].filter((el) => el.getAttribute("style")?.includes("var(--slider-fill)"))
    expect(wired.length).toBe(4)
  })

  it("toggles fullscreen through the callback", async () => {
    const onToggleFullscreen = vi.fn()
    renderControls({ onToggleFullscreen })
    await userEvent.click(screen.getByRole("button", { name: "Tela cheia" }))
    expect(onToggleFullscreen).toHaveBeenCalledOnce()
  })

  it("opens the quality menu and selects a resolution", async () => {
    const onSelectQuality = vi.fn()
    const onMenuOpenChange = vi.fn()
    renderControls({
      onSelectQuality,
      onMenuOpenChange,
      qualities: [
        { label: "Original", variant: "original" },
        { label: "4K", variant: "2160p" },
        { label: "1080p", variant: "1080p" },
      ],
    })

    await userEvent.click(screen.getByRole("button", { name: "Qualidade" }))
    expect(onMenuOpenChange).toHaveBeenLastCalledWith(true)

    const menu = screen.getByRole("menu")
    expect(within(menu).getAllByRole("menuitemradio")).toHaveLength(3)
    const active = within(menu).getByRole("menuitemradio", { name: /Original/ })
    expect(active).toHaveAttribute("aria-checked", "true")

    await userEvent.click(within(menu).getByRole("menuitemradio", { name: /4K/ }))
    expect(onSelectQuality).toHaveBeenCalledWith("2160p")
  })

  it("routes previous and next to their callbacks", async () => {
    const onPrevious = vi.fn()
    const onNext = vi.fn()
    renderControls({ onPrevious, onNext })
    await userEvent.click(screen.getByRole("button", { name: "Episódio anterior" }))
    await userEvent.click(screen.getByRole("button", { name: "Próximo episódio" }))
    expect(onPrevious).toHaveBeenCalledOnce()
    expect(onNext).toHaveBeenCalledOnce()
  })

  it("omits prev and next buttons when no handlers are given", () => {
    renderControls({ onPrevious: undefined, onNext: undefined })
    expect(screen.queryByRole("button", { name: "Episódio anterior" })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "Próximo episódio" })).not.toBeInTheDocument()
  })
})
