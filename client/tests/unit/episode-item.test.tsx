import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { EpisodeItem } from "../../src/components/anime/episode-item"
import type { Episode } from "../../src/api/types"

const mockEpisode: Episode = {
  id: "ep-1",
  title: "Episódio 1",
  number: "1",
  seasonNumber: 1,
  type: "episode",
  url: "https://example.com/ep-1",
}

describe("EpisodeItem", () => {
  it("renders episode title", () => {
    render(<EpisodeItem episode={mockEpisode} checked={false} onToggle={() => {}} />)
    expect(screen.getByText("Episódio 1")).toBeInTheDocument()
  })

  it("renders season and episode number for episode type", () => {
    render(<EpisodeItem episode={mockEpisode} checked={false} onToggle={() => {}} />)
    expect(screen.getByText("1x1")).toBeInTheDocument()
  })

  it("calls onToggle with episode url when clicked", async () => {
    const onToggle = vi.fn()
    const user = userEvent.setup()
    render(<EpisodeItem episode={mockEpisode} checked={false} onToggle={onToggle} />)
    await user.click(screen.getByRole("checkbox"))
    expect(onToggle).toHaveBeenCalledWith("https://example.com/ep-1")
  })

  it("renders as checked when checked prop is true", () => {
    render(<EpisodeItem episode={mockEpisode} checked={true} onToggle={() => {}} />)
    expect(screen.getByRole("checkbox")).toBeChecked()
  })
})
