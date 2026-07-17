import { describe, expect, it } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { http, HttpResponse } from "msw"
import App from "../../src/App"
import { server } from "../../src/mocks/server"

function sample(id: string, klass: string) {
  return {
    id,
    source: "teacher",
    class: klass,
    frameUrl: `/fake/frames/${id}.jpg`,
    maskUrl: `/fake/masks/${id}.png`,
    animeTitle: "Slayers",
    episode: "S1E04",
    timestampS: 54.3,
    teacherProb: 0.42,
    status: "pending",
    createdAt: "2026-07-16",
  }
}

function mockDatasetApi(queue: ReturnType<typeof sample>[]) {
  const verdicts: Record<string, string> = {}
  server.use(
    http.get("/api/dataset/samples/queue", () => {
      return HttpResponse.json(queue.filter((item) => !verdicts[item.id]))
    }),
    http.post("/api/dataset/samples/:id/verdict", async ({ params, request }) => {
      const body = (await request.json()) as { verdict: string }
      verdicts[String(params.id)] = body.verdict
      return new HttpResponse(null, { status: 204 })
    }),
    http.get("/api/dataset/stats", () => {
      return HttpResponse.json({
        total: queue.length,
        pending: queue.length - Object.keys(verdicts).length,
        approved: Object.values(verdicts).filter((v) => v === "approved").length,
        rejected: Object.values(verdicts).filter((v) => v === "rejected").length,
        needsEdit: Object.values(verdicts).filter((v) => v === "needs_edit").length,
        byClass: [],
      })
    }),
  )
  return verdicts
}

async function openDatasetPage(user: ReturnType<typeof userEvent.setup>) {
  render(<App />)
  await user.click(await screen.findByRole("button", { name: "Dataset" }))
  await screen.findByRole("heading", { level: 1, name: "Triagem do Dataset" })
}

describe("Dataset Triage Page", () => {
  it("shows the current sample with class badge and mask overlay", async () => {
    mockDatasetApi([sample("1", "fire")])
    const user = userEvent.setup()
    await openDatasetPage(user)

    expect(await screen.findByText("Fogo")).toBeInTheDocument()
    expect(screen.getByText(/Slayers/)).toBeInTheDocument()
    expect(screen.getByTestId("mask-overlay")).toBeInTheDocument()
  })

  it("approves via button and advances to the next sample", async () => {
    const verdicts = mockDatasetApi([sample("1", "fire"), sample("2", "lightning")])
    const user = userEvent.setup()
    await openDatasetPage(user)

    await screen.findByText("Fogo")
    await user.click(screen.getByRole("button", { name: /Aprovar/ }))

    expect(await screen.findByText("Relâmpago")).toBeInTheDocument()
    expect(verdicts["1"]).toBe("approved")
  })

  it("submits verdicts with keyboard shortcuts", async () => {
    const verdicts = mockDatasetApi([sample("1", "fire"), sample("2", "aura")])
    const user = userEvent.setup()
    await openDatasetPage(user)

    await screen.findByText("Fogo")
    await user.keyboard("r")

    expect(await screen.findByText("Aura")).toBeInTheDocument()
    expect(verdicts["1"]).toBe("rejected")

    await user.keyboard("e")
    await waitFor(() => {
      expect(verdicts["2"]).toBe("needs_edit")
    })
  })

  it("toggles the mask overlay with the M key", async () => {
    mockDatasetApi([sample("1", "fire")])
    const user = userEvent.setup()
    await openDatasetPage(user)

    await screen.findByTestId("mask-overlay")
    await user.keyboard("m")
    await waitFor(() => {
      expect(screen.queryByTestId("mask-overlay")).not.toBeInTheDocument()
    })
  })

  it("shows the empty state when the queue is empty", async () => {
    mockDatasetApi([])
    const user = userEvent.setup()
    await openDatasetPage(user)

    expect(await screen.findByText(/Fila vazia/)).toBeInTheDocument()
  })
})
