import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import { AggressiveUpscaleWarning } from "../../src/pages/upscale"
import type { Episode } from "../../src/api/types"

function episode(id: string): Episode {
  return {
    id,
    title: `Episódio ${id}`,
    number: id,
    seasonNumber: 1,
    url: `https://example.com/ep-${id}`,
  } as Episode
}

describe("AggressiveUpscaleWarning", () => {
  it("renders nothing when no source resolution is known", () => {
    render(
      <AggressiveUpscaleWarning
        episodes={[episode("1")]}
        heights={new Map()}
        targetHeight={2160}
      />,
    )
    expect(screen.queryByRole("status")).not.toBeInTheDocument()
  })

  it("renders nothing when the upscale factor is safe", () => {
    render(
      <AggressiveUpscaleWarning
        episodes={[episode("1")]}
        heights={new Map([["1", 480]])}
        targetHeight={1080}
      />,
    )
    expect(screen.queryByRole("status")).not.toBeInTheDocument()
  })

  it("warns with a safer suggestion for degraded sources", () => {
    render(
      <AggressiveUpscaleWarning
        episodes={[episode("1")]}
        heights={new Map([["1", 480]])}
        targetHeight={2160}
      />,
    )
    const warning = screen.getByRole("status")
    expect(warning).toHaveTextContent("Upscale agressivo demais")
    expect(warning).toHaveTextContent("~480 linhas")
    expect(warning).toHaveTextContent("Recomendado: 1440p.")
  })

  it("counts only the risky episodes and uses the lowest source", () => {
    render(
      <AggressiveUpscaleWarning
        episodes={[episode("1"), episode("2"), episode("3")]}
        heights={new Map([["1", 240], ["2", 330], ["3", 1080]])}
        targetHeight={1080}
      />,
    )
    const warning = screen.getByRole("status")
    expect(warning).toHaveTextContent("2 episódios")
    expect(warning).toHaveTextContent("~240 linhas")
  })

  it("recommends keeping a lower resolution when nothing is safe", () => {
    render(
      <AggressiveUpscaleWarning
        episodes={[episode("1")]}
        heights={new Map([["1", 240]])}
        targetHeight={1080}
      />,
    )
    expect(screen.getByRole("status")).toHaveTextContent("Recomendado manter uma resolução menor.")
  })
})
