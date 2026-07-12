import { QueryClient, QueryClientProvider, QueryCache, MutationCache } from "@tanstack/react-query"
import { Toaster, toast } from "sonner"
import { AppShell } from "@/components/layout/app-shell"
import { useRoute, type PageRoute } from "@/hooks/use-route"
import { DownloadsPage } from "@/pages/downloads"
import { CatalogPage } from "@/pages/catalog"
import { EditionPage } from "@/pages/upscale"
import { ComparePage } from "@/pages/compare"
import { LoginPage } from "@/pages/login"
import { InvitesPage } from "@/pages/invites"

function errorDescription(error: Error): string {
  const text = error.message.trim()
  if (!text) return "Erro inesperado. Tente novamente."
  return text.length > 140 ? `${text.slice(0, 140)}…` : text
}

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      staleTime: 5 * 60 * 1000,
    },
  },
  queryCache: new QueryCache({
    onError: (error, query) => {
      if (query.meta?.silentError) return
      toast.error("Falha ao carregar dados", {
        id: `query-${query.queryHash}`,
        description: errorDescription(error),
      })
    },
  }),
  mutationCache: new MutationCache({
    onError: (error) => {
      toast.error("A operação falhou", { description: errorDescription(error) })
    },
  }),
})

function PageRouter({ page }: { page: PageRoute }) {
  if (page === "catalog") return <CatalogPage />
  if (page === "upscale") return <EditionPage />
  if (page === "compare") return <ComparePage />
  if (page === "invites") return <InvitesPage />
  return <DownloadsPage />
}

export default function App() {
  const { page, navigate } = useRoute()

  return (
    <QueryClientProvider client={queryClient}>
      {page === "login" ? (
        <LoginPage />
      ) : (
        <AppShell currentPage={page} onNavigate={navigate}>
          <PageRouter page={page} />
        </AppShell>
      )}
      <Toaster theme="dark" position="bottom-right" richColors closeButton />
    </QueryClientProvider>
  )
}
