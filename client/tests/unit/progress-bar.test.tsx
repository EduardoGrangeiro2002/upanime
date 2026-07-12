import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { ProgressBar } from "../../src/components/download/progress-bar"

describe("ProgressBar", () => {
  it("renders with 0% width", () => {
    render(<ProgressBar progress={0} />)
    const bar = screen.getByRole("progressbar")
    expect(bar).toHaveAttribute("aria-valuenow", "0")
    expect(bar.style.width).toBe("0%")
  })

  it("renders with 50% width", () => {
    render(<ProgressBar progress={50} />)
    const bar = screen.getByRole("progressbar")
    expect(bar).toHaveAttribute("aria-valuenow", "50")
    expect(bar.style.width).toBe("50%")
  })

  it("renders with 100% width", () => {
    render(<ProgressBar progress={100} />)
    const bar = screen.getByRole("progressbar")
    expect(bar).toHaveAttribute("aria-valuenow", "100")
    expect(bar.style.width).toBe("100%")
  })

  it("clamps values above 100", () => {
    render(<ProgressBar progress={150} />)
    const bar = screen.getByRole("progressbar")
    expect(bar).toHaveAttribute("aria-valuenow", "100")
  })

  it("clamps negative values to 0", () => {
    render(<ProgressBar progress={-10} />)
    const bar = screen.getByRole("progressbar")
    expect(bar).toHaveAttribute("aria-valuenow", "0")
  })
})
