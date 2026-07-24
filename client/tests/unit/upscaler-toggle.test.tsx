import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { ProcessingConfig } from "../../src/pages/upscale"

function renderConfig(upscaler: "compact" | "apisr", onUpscalerChange = vi.fn()) {
  render(
    <ProcessingConfig
      targetHeight={1440}
      onChange={vi.fn()}
      interpolate={false}
      onInterpolateChange={vi.fn()}
      panRatio={0.6}
      onPanRatioChange={vi.fn()}
      effects={false}
      onEffectsChange={vi.fn()}
      effectsParams={{ strength: 1.0, sensitivity: 1.0 }}
      onEffectsParamsChange={vi.fn()}
      skipUpscale={false}
      onSkipUpscaleChange={vi.fn()}
      upscaler={upscaler}
      onUpscalerChange={onUpscalerChange}
      encodeParams={{ batchSize: 2, sharpen: 0.5, saturation: 1.2, contrast: 1.05 }}
      onEncodeParamsChange={vi.fn()}
    />,
  )
  return onUpscalerChange
}

describe("ProcessingConfig upscaler toggle", () => {
  it("activates apisr when toggled on", async () => {
    const onChange = renderConfig("compact")
    await userEvent.click(screen.getByRole("button", { name: "Desativado" }))
    expect(onChange).toHaveBeenCalledWith("apisr")
  })

  it("returns to compact when toggled off", async () => {
    const onChange = renderConfig("apisr")
    await userEvent.click(screen.getByRole("button", { name: "Ativado" }))
    expect(onChange).toHaveBeenCalledWith("compact")
  })
})
