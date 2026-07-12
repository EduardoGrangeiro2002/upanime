import { useState, type FormEvent } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { Loader2, Mail, ShieldCheck, UserPlus, Users } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { fetchUsers, sendInvite } from "@/api/endpoints"
import { useMe } from "@/hooks/use-me"
import type { UserSummary } from "@/api/types"

export function InvitesPage() {
  const { data: me } = useMe()

  if (me && !me.isAdmin) {
    return (
      <div className="px-4 py-16 text-center space-y-2">
        <ShieldCheck className="h-10 w-10 mx-auto text-muted-foreground opacity-40" aria-hidden="true" />
        <p className="text-sm text-muted-foreground">Esta área é restrita a administradores.</p>
      </div>
    )
  }

  return (
    <div className="space-y-8 px-4 py-4 md:px-8 md:py-8 max-w-3xl mx-auto">
      <HeroSection />
      <InviteForm />
      <UserList />
    </div>
  )
}

function HeroSection() {
  return (
    <div className="relative overflow-hidden rounded-2xl bg-surface-high p-4 md:p-8">
      <div className="absolute top-4 right-4 opacity-10" aria-hidden="true">
        <UserPlus className="h-32 w-32 text-primary" />
      </div>
      <div className="relative z-10 max-w-lg space-y-3">
        <h1 className="font-display text-2xl md:text-4xl font-bold tracking-tighter">Convites</h1>
        <p className="text-muted-foreground text-sm leading-relaxed">
          Convide alguém pelo email: a conta é criada com uma senha temporária enviada por email, e a troca é exigida no primeiro login.
        </p>
      </div>
    </div>
  )
}

function InviteForm() {
  const [email, setEmail] = useState("")
  const queryClient = useQueryClient()

  const invite = useMutation({
    mutationFn: sendInvite,
    onSuccess: (created) => {
      queryClient.invalidateQueries({ queryKey: ["users"] })
      toast.success(`Convite enviado para ${created.email}`)
      setEmail("")
    },
  })

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    if (!email.trim()) return
    invite.mutate(email.trim())
  }

  return (
    <form onSubmit={handleSubmit} className="rounded-xl glass p-4 md:p-6 space-y-3">
      <label htmlFor="invite-email" className="flex items-center gap-1.5 text-sm font-medium">
        <Mail className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
        Email do convidado
      </label>
      <div className="flex gap-2">
        <Input
          id="invite-email"
          type="email"
          required
          placeholder="pessoa@exemplo.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <Button type="submit" variant="gradient" disabled={invite.isPending}>
          {invite.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          ) : (
            <UserPlus className="h-4 w-4" aria-hidden="true" />
          )}
          Convidar
        </Button>
      </div>
    </form>
  )
}

function UserList() {
  const { data: users } = useQuery({ queryKey: ["users"], queryFn: fetchUsers })

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Users className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
        <h2 className="font-display text-xl font-bold">Usuários</h2>
        {users && users.length > 0 && (
          <Badge variant="secondary" className="text-[10px]">{users.length}</Badge>
        )}
      </div>

      {!users || users.length === 0 ? (
        <div className="rounded-xl bg-surface p-8 text-center opacity-60">
          <Users className="h-10 w-10 mx-auto mb-3 text-muted-foreground opacity-40" aria-hidden="true" />
          <p className="text-sm text-muted-foreground">Nenhum usuário ainda</p>
        </div>
      ) : (
        <ul className="space-y-2">
          {users.map((user) => (
            <UserRow key={user.email} user={user} />
          ))}
        </ul>
      )}
    </div>
  )
}

function UserRow({ user }: { user: UserSummary }) {
  return (
    <li className="flex items-center justify-between gap-3 rounded-xl bg-surface p-3">
      <span className="text-sm truncate" title={user.email}>{user.email}</span>
      <span className="flex items-center gap-1.5 shrink-0">
        {user.isAdmin && (
          <Badge variant="secondary" className="text-[10px]">Admin</Badge>
        )}
        <Badge variant={user.pending ? "outline" : "success"} className="text-[10px]">
          {user.pending ? "Pendente" : "Ativo"}
        </Badge>
      </span>
    </li>
  )
}
