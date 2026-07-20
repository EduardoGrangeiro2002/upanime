import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import { MediaPlayer, MediaProvider } from "@vidstack/react"
import { PlayerControls } from "../../src/components/catalog/player-controls"

function renderControls(overrides: Partial<Parameters<typeof PlayerControls>[0]> = {}) {
  return render(
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
        menuOpen={false}
        onMenuOpenChange={() => {}}
        {...overrides}
      />
    </MediaPlayer>,
  )
}

describe("PlayerControls", () => {
  it("renders the pt-BR control labels", () => {
    renderControls()
    for (const label of ["Fechar player", "Episódio anterior", "Próximo episódio", "Avançar 10 segundos", "Voltar 10 segundos", "Qualidade", "Tela cheia"]) {
      expect(screen.getByRole("button", { name: label })).toBeInTheDocument()
    }
  })

  it("shows the exit label when already in fullscreen", () => {
    renderControls({ isFullscreen: true })
    expect(screen.getByRole("button", { name: "Sair da tela cheia" })).toBeInTheDocument()
  })

  it("hides the quality gear when there is a single option", () => {
    renderControls({ qualities: [{ label: "Original", variant: "original" }] })
    expect(screen.queryByRole("button", { name: "Qualidade" })).not.toBeInTheDocument()
  })
})
