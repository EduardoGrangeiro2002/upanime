import { useState, type FormEvent } from "react"
import { KeyRound, Loader2, Mail, ShieldCheck } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { authChangePassword, authForgot, authLogin, authReset, authVerifyMfa } from "@/api/endpoints"
import { ApiError } from "@/api/client"
import type { AuthStep } from "@/api/types"

type AuthView = "login" | "change_password" | "mfa" | "forgot" | "reset"

const VIEW_TITLES: Record<AuthView, string> = {
  login: "Entrar",
  change_password: "Definir nova senha",
  mfa: "Verificação de acesso",
  forgot: "Recuperar senha",
  reset: "Redefinir senha",
}

const VIEW_HINTS: Record<AuthView, string> = {
  login: "Acesso somente por convite — não há registro público.",
  change_password: "Sua senha temporária precisa ser substituída antes de continuar.",
  mfa: "Enviamos um código de 6 dígitos para o seu email. Ele expira em 15 minutos.",
  forgot: "Informe seu email e enviaremos um código de redefinição.",
  reset: "Digite o código recebido por email e a nova senha.",
}

export function LoginPage() {
  const [view, setView] = useState<AuthView>("login")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [code, setCode] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [info, setInfo] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  const enterApp = () => {
    window.location.hash = "#/downloads"
  }

  const followStep = (step: AuthStep) => {
    if (step === "change_password") {
      setView("change_password")
      return
    }
    if (step === "mfa") {
      setView("mfa")
      return
    }
    enterApp()
  }

  const run = async (action: () => Promise<void>) => {
    setError(null)
    setInfo(null)
    setPending(true)
    try {
      await action()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(parseErrorMessage(err.message))
        return
      }
      setError("Erro inesperado. Tente novamente.")
    } finally {
      setPending(false)
    }
  }

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()

    if (view === "login") {
      run(async () => {
        const { step } = await authLogin(email, password)
        followStep(step)
      })
      return
    }

    if (view === "change_password") {
      run(async () => {
        const { step } = await authChangePassword(email, password, newPassword)
        setPassword(newPassword)
        followStep(step)
      })
      return
    }

    if (view === "mfa") {
      run(async () => {
        const { step } = await authVerifyMfa(email, code)
        followStep(step)
      })
      return
    }

    if (view === "forgot") {
      run(async () => {
        await authForgot(email)
        setInfo("Se o email existir, um código foi enviado.")
        setView("reset")
      })
      return
    }

    run(async () => {
      await authReset(email, code, newPassword)
      setInfo("Senha redefinida. Entre com a nova senha.")
      setPassword("")
      setNewPassword("")
      setCode("")
      setView("login")
    })
  }

  return (
    <main className="min-h-screen flex items-center justify-center px-4 bg-background">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center space-y-2">
          <p className="font-display text-2xl font-bold bg-gradient-to-r from-primary to-primary-dim bg-clip-text text-transparent">
            UpAnime
          </p>
          <h1 className="font-display text-xl font-bold">{VIEW_TITLES[view]}</h1>
          <p className="text-xs text-muted-foreground">{VIEW_HINTS[view]}</p>
        </div>

        <form onSubmit={handleSubmit} className="rounded-2xl glass border border-white/[0.08] p-6 space-y-4">
          {(view === "login" || view === "forgot") && (
            <Field label="Email" id="auth-email" icon={<Mail className="h-3.5 w-3.5" aria-hidden="true" />}>
              <Input
                id="auth-email"
                type="email"
                required
                autoComplete="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </Field>
          )}

          {view === "login" && (
            <Field label="Senha" id="auth-password" icon={<KeyRound className="h-3.5 w-3.5" aria-hidden="true" />}>
              <Input
                id="auth-password"
                type="password"
                required
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </Field>
          )}

          {(view === "mfa" || view === "reset") && (
            <Field label="Código" id="auth-code" icon={<ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />}>
              <Input
                id="auth-code"
                inputMode="numeric"
                pattern="[0-9]{6}"
                maxLength={6}
                required
                autoComplete="one-time-code"
                value={code}
                onChange={(e) => setCode(e.target.value)}
              />
            </Field>
          )}

          {(view === "change_password" || view === "reset") && (
            <Field label="Nova senha" id="auth-new-password" icon={<KeyRound className="h-3.5 w-3.5" aria-hidden="true" />}>
              <Input
                id="auth-new-password"
                type="password"
                required
                minLength={8}
                autoComplete="new-password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
              />
            </Field>
          )}

          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
          {info && (
            <p role="status" className="text-sm text-muted-foreground">
              {info}
            </p>
          )}

          <Button type="submit" variant="gradient" className="w-full" disabled={pending}>
            {pending && <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />}
            {VIEW_TITLES[view]}
          </Button>

          <div className="flex justify-between text-xs">
            {view === "login" ? (
              <button
                type="button"
                onClick={() => {
                  setError(null)
                  setInfo(null)
                  setView("forgot")
                }}
                className="text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
              >
                Esqueci minha senha
              </button>
            ) : (
              <button
                type="button"
                onClick={() => {
                  setError(null)
                  setInfo(null)
                  setView("login")
                }}
                className="text-muted-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
              >
                Voltar ao login
              </button>
            )}
          </div>
        </form>
      </div>
    </main>
  )
}

function Field({ label, id, icon, children }: { label: string; id: string; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={id} className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
        {icon}
        {label}
      </label>
      {children}
    </div>
  )
}

function parseErrorMessage(raw: string): string {
  try {
    const parsed = JSON.parse(raw) as { error?: string }
    if (parsed.error) return parsed.error
  } catch {
    return raw || "Erro inesperado. Tente novamente."
  }
  return raw || "Erro inesperado. Tente novamente."
}
