import "@testing-library/jest-dom/vitest"
import { beforeAll, afterEach, afterAll } from "vitest"
import { cleanup } from "@testing-library/react"
import { server } from "../src/mocks/server"
import { resetMockState } from "../src/mocks/handlers"
import { queryClient } from "../src/App"

class IntersectionObserverMock {
  disconnect() {}
  observe() {}
  unobserve() {}
  takeRecords() {
    return []
  }
}

class ResizeObserverMock {
  disconnect() {}
  observe() {}
  unobserve() {}
}

Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    addListener: () => undefined,
    removeListener: () => undefined,
    dispatchEvent: () => false,
  }),
})

Object.defineProperty(window, "IntersectionObserver", {
  writable: true,
  value: IntersectionObserverMock,
})

Object.defineProperty(window, "ResizeObserver", {
  writable: true,
  value: ResizeObserverMock,
})

beforeAll(() => server.listen({ onUnhandledRequest: "error" }))
afterEach(() => {
  resetMockState()
  server.resetHandlers()
  cleanup()
  queryClient.clear()
  window.history.replaceState(null, "", window.location.pathname)
  document.body.style.overflow = ""
})
afterAll(() => server.close())
