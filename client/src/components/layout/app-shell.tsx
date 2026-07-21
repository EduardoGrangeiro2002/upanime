import type { ReactNode } from "react"
import { Navbar } from "./navbar"
import type { PageRoute } from "@/hooks/use-route"

interface AppShellProps {
  currentPage: PageRoute
  onNavigate: (page: PageRoute) => void
  children: ReactNode
}

export function AppShell({ currentPage, onNavigate, children }: AppShellProps) {
  return (
    <div className="min-h-screen bg-background">
      <Navbar currentPage={currentPage} onNavigate={onNavigate} />
      <main className="min-h-screen pt-16 pb-20 md:pb-0">{children}</main>
    </div>
  )
}
