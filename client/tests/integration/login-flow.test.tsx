import { describe, expect, it } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { http, HttpResponse } from "msw"
import App from "../../src/App"
import { server } from "../../src/mocks/server"

async function openLoginPage() {
  window.location.hash = "#/login"
  render(<App />)
  await screen.findByRole("heading", { name: "Entrar" })
}

describe("Login flow", () => {
  it("has no public registration path", async () => {
    await openLoginPage()

    expect(screen.queryByRole("button", { name: /registr|criar conta|cadastr/i })).not.toBeInTheDocument()
    expect(screen.queryByRole("link", { name: /registr|criar conta|cadastr/i })).not.toBeInTheDocument()
    expect(screen.getByText(/somente por convite/i)).toBeInTheDocument()
  })

  it("completes first login: temp password, change, mfa, then enters the app", async () => {
    const user = userEvent.setup()
    await openLoginPage()

    await user.type(screen.getByLabelText("Email"), "dono@upanime.dev")
    await user.type(screen.getByLabelText("Senha"), "senha-temporaria")
    await user.click(screen.getByRole("button", { name: "Entrar" }))

    await screen.findByRole("heading", { name: "Definir nova senha" })
    await user.type(screen.getByLabelText("Nova senha"), "senha-definitiva")
    await user.click(screen.getByRole("button", { name: "Definir nova senha" }))

    await screen.findByRole("heading", { name: "Verificação de acesso" })
    await user.type(screen.getByLabelText("Código"), "123456")
    await user.click(screen.getByRole("button", { name: "Verificação de acesso" }))

    await waitFor(() => {
      expect(window.location.hash).toBe("#/downloads")
    })
  })

  it("shows an error and stays on login for wrong credentials", async () => {
    const user = userEvent.setup()
    await openLoginPage()

    await user.type(screen.getByLabelText("Email"), "dono@upanime.dev")
    await user.type(screen.getByLabelText("Senha"), "senha-errada")
    await user.click(screen.getByRole("button", { name: "Entrar" }))

    const alert = await screen.findByRole("alert")
    expect(alert).toHaveTextContent(/inválidos/i)
    expect(screen.getByRole("heading", { name: "Entrar" })).toBeInTheDocument()
  })

  it("rejects a wrong mfa code", async () => {
    const user = userEvent.setup()
    await openLoginPage()

    await user.type(screen.getByLabelText("Email"), "dono@upanime.dev")
    await user.type(screen.getByLabelText("Senha"), "senha-valida")
    await user.click(screen.getByRole("button", { name: "Entrar" }))

    await screen.findByRole("heading", { name: "Verificação de acesso" })
    await user.type(screen.getByLabelText("Código"), "999999")
    await user.click(screen.getByRole("button", { name: "Verificação de acesso" }))

    const alert = await screen.findByRole("alert")
    expect(alert).toHaveTextContent(/código inválido/i)
  })

  it("recovers access through the forgot password flow", async () => {
    const user = userEvent.setup()
    await openLoginPage()

    await user.click(screen.getByRole("button", { name: "Esqueci minha senha" }))
    await screen.findByRole("heading", { name: "Recuperar senha" })

    await user.type(screen.getByLabelText("Email"), "dono@upanime.dev")
    await user.click(screen.getByRole("button", { name: "Recuperar senha" }))

    await screen.findByRole("heading", { name: "Redefinir senha" })
    await user.type(screen.getByLabelText("Código"), "123456")
    await user.type(screen.getByLabelText("Nova senha"), "senha-nova-123")
    await user.click(screen.getByRole("button", { name: "Redefinir senha" }))

    await screen.findByRole("heading", { name: "Entrar" })
    expect(screen.getByRole("status")).toHaveTextContent(/senha redefinida/i)
  })

  it("redirects to login when the API returns 401", async () => {
    server.use(
      http.get("/api/downloads", () => {
        return HttpResponse.json({ error: "não autenticado" }, { status: 401 })
      }),
    )

    render(<App />)

    await waitFor(() => {
      expect(window.location.hash).toBe("#/login")
    })
    await screen.findByRole("heading", { name: "Entrar" })
  })

  it("logs out from the navbar back to the login page", async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.click(await screen.findByRole("button", { name: "Sair" }))

    await waitFor(() => {
      expect(window.location.hash).toBe("#/login")
    })
    await screen.findByRole("heading", { name: "Entrar" })
  })
})
