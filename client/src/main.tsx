import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import "./index.css"
import App from "./App.tsx"

async function bootstrap() {
  if (import.meta.env.VITE_MOCK === "true") {
    const { worker } = await import("./mocks/browser.ts")
    await worker.start({ onUnhandledRequest: "bypass" })
    const { seedDownloads } = await import("./mocks/seed-downloads.ts")
    seedDownloads()
  }
  createRoot(document.getElementById("root")!).render(
    <StrictMode>
      <App />
    </StrictMode>,
  )
}

bootstrap()
