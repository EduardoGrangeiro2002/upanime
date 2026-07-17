import { useCallback, useEffect, useState } from "react"

export type PageRoute = "downloads" | "catalog" | "upscale" | "compare" | "login" | "invites" | "dataset"

const PATH_TO_PAGE: Record<string, PageRoute> = {
  downloads: "downloads",
  catalogo: "catalog",
  upscale: "upscale",
  comparar: "compare",
  login: "login",
  convites: "invites",
  dataset: "dataset",
}

const PAGE_TO_PATH: Record<PageRoute, string> = {
  downloads: "downloads",
  catalog: "catalogo",
  upscale: "upscale",
  compare: "comparar",
  login: "login",
  invites: "convites",
  dataset: "dataset",
}

export interface Route {
  page: PageRoute
  param: string | null
}

export function parseHash(hash: string): Route {
  const segments = hash.replace(/^#\/?/, "").split("/").filter(Boolean)
  const page = PATH_TO_PAGE[segments[0]] ?? "downloads"
  const param = segments[1] ? decodeURIComponent(segments[1]) : null
  return { page, param }
}

export function buildHash(page: PageRoute, param?: string | null): string {
  const base = `#/${PAGE_TO_PATH[page]}`
  if (!param) return base
  return `${base}/${encodeURIComponent(param)}`
}

export function useRoute() {
  const [route, setRoute] = useState<Route>(() => parseHash(window.location.hash))

  useEffect(() => {
    const onHashChange = () => setRoute(parseHash(window.location.hash))
    window.addEventListener("hashchange", onHashChange)
    return () => window.removeEventListener("hashchange", onHashChange)
  }, [])

  const navigate = useCallback((page: PageRoute, param?: string | null) => {
    window.location.hash = buildHash(page, param)
  }, [])

  return { ...route, navigate }
}
