import { Download, Library, Wand2, SplitSquareHorizontal, LogOut, UserPlus, Database } from "lucide-react"
import { cn } from "@/lib/utils"
import { authLogout } from "@/api/endpoints"
import { useMe } from "@/hooks/use-me"
import type { PageRoute } from "@/hooks/use-route"

interface NavbarProps {
  currentPage: PageRoute
  onNavigate: (page: PageRoute) => void
}

const baseNavItems: { route: PageRoute; icon: typeof Download; label: string }[] = [
  { route: "downloads", icon: Download, label: "Downloads" },
  { route: "catalog", icon: Library, label: "Catálogo" },
  { route: "upscale", icon: Wand2, label: "Upscale" },
  { route: "compare", icon: SplitSquareHorizontal, label: "Comparar" },
  { route: "dataset", icon: Database, label: "Dataset" },
]

export function Navbar({ currentPage, onNavigate }: NavbarProps) {
  const { data: me } = useMe()
  const navItems = me?.isAdmin
    ? [...baseNavItems, { route: "invites" as PageRoute, icon: UserPlus, label: "Convites" }]
    : baseNavItems

  return (
    <>
      <header className="fixed top-0 left-0 right-0 z-40 h-16 glass border-b border-white/[0.06]">
        <div className="flex h-full items-center justify-between px-4 md:px-8">
          <button
            onClick={() => onNavigate("downloads")}
            aria-label="UpAnime — início"
            className="font-display text-lg md:text-xl font-bold bg-gradient-to-r from-primary to-primary-dim bg-clip-text text-transparent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-md"
          >
            UpAnime
          </button>

          <button
            onClick={async () => {
              await authLogout().catch(() => undefined)
              window.location.hash = "#/login"
            }}
            aria-label="Sair"
            data-tooltip="Sair"
            data-tooltip-pos="left"
            className="flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground hover:text-foreground hover:bg-surface-high transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <LogOut className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
      </header>

      <nav
        aria-label="Navegação principal"
        className="fixed bottom-0 left-0 right-0 z-40 flex items-stretch justify-around glass border-t border-white/[0.06] px-1 pb-[env(safe-area-inset-bottom)] md:bottom-auto md:top-0 md:left-1/2 md:right-auto md:-translate-x-1/2 md:h-16 md:items-center md:justify-center md:gap-1 md:border-t-0 md:bg-transparent md:backdrop-blur-none md:px-0 md:pb-0"
      >
        {navItems.map((item) => (
          <NavLink
            key={item.route}
            icon={item.icon}
            label={item.label}
            active={currentPage === item.route}
            onClick={() => onNavigate(item.route)}
          />
        ))}
      </nav>
    </>
  )
}

interface NavLinkProps {
  icon: typeof Download
  label: string
  active: boolean
  onClick: () => void
}

function NavLink({ icon: Icon, label, active, onClick }: NavLinkProps) {
  return (
    <button
      onClick={onClick}
      aria-current={active ? "page" : undefined}
      className={cn(
        "relative flex flex-1 md:flex-none flex-col md:flex-row items-center gap-1 md:gap-2 rounded-lg px-1 md:px-4 py-2.5 md:py-2 text-[10px] md:text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        active
          ? "text-primary"
          : "text-muted-foreground hover:text-foreground",
      )}
    >
      <Icon className="h-5 w-5 md:h-4 md:w-4 shrink-0" aria-hidden="true" />
      <span>{label}</span>
      {active && (
        <span className="hidden md:block absolute bottom-0 left-3 right-3 h-[2px] rounded-full bg-primary neon-primary" />
      )}
    </button>
  )
}
