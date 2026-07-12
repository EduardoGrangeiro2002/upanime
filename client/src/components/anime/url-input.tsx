import { useState } from "react"
import { Search, Loader2 } from "lucide-react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"

interface UrlInputProps {
  onSubmit: (url: string) => void
  isLoading: boolean
}

function validateUrl(value: string): string | null {
  try {
    const parsed = new URL(value)
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
      return "A URL precisa começar com http:// ou https://."
    }
    return null
  } catch {
    return "Isso não parece uma URL. Cole o endereço completo da página do anime."
  }
}

export function UrlInput({ onSubmit, isLoading }: UrlInputProps) {
  const [url, setUrl] = useState("")
  const [validationError, setValidationError] = useState<string | null>(null)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = url.trim()
    if (!trimmed) return
    const error = validateUrl(trimmed)
    if (error) {
      setValidationError(error)
      return
    }
    setValidationError(null)
    onSubmit(trimmed)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-2">
      <div className="flex gap-2">
        <Input
          value={url}
          onChange={(e) => {
            setUrl(e.target.value)
            if (validationError) setValidationError(null)
          }}
          placeholder="https://animesonlinecc.to/anime/shingeki-no-kyojin/"
          aria-label="URL do anime"
          aria-invalid={validationError !== null}
          aria-describedby={validationError ? "url-input-error" : undefined}
          className="flex-1 glass"
          disabled={isLoading}
        />
        <Button type="submit" variant="gradient" disabled={isLoading || !url.trim()}>
          {isLoading ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Search className="h-4 w-4" aria-hidden="true" />}
          Buscar
        </Button>
      </div>
      {validationError && (
        <p id="url-input-error" role="alert" className="text-xs text-destructive px-1">
          {validationError}
        </p>
      )}
    </form>
  )
}
