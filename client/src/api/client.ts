const API_BASE = "/api"

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  })

  if (!res.ok) {
    if (res.status === 401 && !path.startsWith("/auth")) {
      window.location.hash = "#/login"
    }
    throw new ApiError(res.status, await res.text())
  }

  if (res.status === 204) {
    return undefined as T
  }

  return res.json()
}
