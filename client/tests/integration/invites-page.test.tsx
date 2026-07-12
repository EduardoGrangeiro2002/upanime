import { describe, expect, it } from "vitest"
import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { http, HttpResponse } from "msw"
import App from "../../src/App"
import { server } from "../../src/mocks/server"

async function openInvitesPage(user: ReturnType<typeof userEvent.setup>) {
  render(<App />)
  await user.click(await screen.findByRole("button", { name: "Convites" }))
  await screen.findByRole("heading", { level: 1, name: "Convites" })
}

describe("Invites Page", () => {
  it("shows the invites nav item only for admins", async () => {
    render(<App />)
    expect(await screen.findByRole("button", { name: "Convites" })).toBeInTheDocument()
  })

  it("hides the invites nav item for non-admins", async () => {
    server.use(
      http.get("/api/auth/me", () => {
        return HttpResponse.json({ email: "comum@upanime.dev", isAdmin: false })
      }),
    )

    render(<App />)

    await screen.findByRole("button", { name: "Downloads" })
    await waitFor(() => {
      expect(screen.queryByRole("button", { name: "Convites" })).not.toBeInTheDocument()
    })
  })

  it("lists existing users with role and status badges", async () => {
    const user = userEvent.setup()
    await openInvitesPage(user)

    const row = (await screen.findByText("admin@upanime.dev")).closest("li")!
    expect(within(row).getByText("Admin")).toBeInTheDocument()
    expect(within(row).getByText("Ativo")).toBeInTheDocument()
  })

  it("invites a user and shows them as pending", async () => {
    const user = userEvent.setup()
    await openInvitesPage(user)

    await user.type(screen.getByLabelText(/email do convidado/i), "novo@upanime.dev")
    await user.click(screen.getByRole("button", { name: /convidar/i }))

    const row = (await screen.findByText("novo@upanime.dev")).closest("li")!
    expect(within(row).getByText("Pendente")).toBeInTheDocument()
    expect(within(row).queryByText("Admin")).not.toBeInTheDocument()
  })

  it("surfaces a conflict error for an already invited email", async () => {
    const user = userEvent.setup()
    await openInvitesPage(user)

    await user.type(screen.getByLabelText(/email do convidado/i), "admin@upanime.dev")
    await user.click(screen.getByRole("button", { name: /convidar/i }))

    await waitFor(() => {
      expect(screen.getAllByText(/já existe/i).length).toBeGreaterThan(0)
    })
  })
})
