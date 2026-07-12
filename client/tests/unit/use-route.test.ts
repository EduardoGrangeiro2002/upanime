import { describe, it, expect } from "vitest"
import { parseHash, buildHash } from "../../src/hooks/use-route"

describe("parseHash", () => {
  it("defaults to downloads when hash is empty", () => {
    expect(parseHash("")).toEqual({ page: "downloads", param: null })
    expect(parseHash("#")).toEqual({ page: "downloads", param: null })
    expect(parseHash("#/")).toEqual({ page: "downloads", param: null })
  })

  it("parses each page path", () => {
    expect(parseHash("#/downloads")).toEqual({ page: "downloads", param: null })
    expect(parseHash("#/catalogo")).toEqual({ page: "catalog", param: null })
    expect(parseHash("#/upscale")).toEqual({ page: "upscale", param: null })
    expect(parseHash("#/comparar")).toEqual({ page: "compare", param: null })
  })

  it("parses a param segment", () => {
    expect(parseHash("#/catalogo/anime-42")).toEqual({ page: "catalog", param: "anime-42" })
  })

  it("decodes encoded params", () => {
    expect(parseHash("#/catalogo/a%20b")).toEqual({ page: "catalog", param: "a b" })
  })

  it("falls back to downloads for unknown paths", () => {
    expect(parseHash("#/nao-existe")).toEqual({ page: "downloads", param: null })
  })
})

describe("buildHash", () => {
  it("builds page hashes", () => {
    expect(buildHash("downloads")).toBe("#/downloads")
    expect(buildHash("catalog")).toBe("#/catalogo")
    expect(buildHash("upscale")).toBe("#/upscale")
    expect(buildHash("compare")).toBe("#/comparar")
  })

  it("builds hashes with params and encodes them", () => {
    expect(buildHash("catalog", "anime-42")).toBe("#/catalogo/anime-42")
    expect(buildHash("catalog", "a b")).toBe("#/catalogo/a%20b")
  })

  it("round-trips with parseHash", () => {
    expect(parseHash(buildHash("catalog", "x/y"))).toEqual({ page: "catalog", param: "x/y" })
  })
})
