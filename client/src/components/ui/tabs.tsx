import * as React from "react"
import { cn } from "@/lib/utils"

interface TabsContextValue {
  value: string
  onValueChange: (value: string) => void
}

const TabsContext = React.createContext<TabsContextValue>({
  value: "",
  onValueChange: () => {},
})

function tabId(value: string) {
  return `tab-${value.replace(/\s+/g, "-")}`
}

function panelId(value: string) {
  return `panel-${value.replace(/\s+/g, "-")}`
}

interface TabsProps extends React.ComponentProps<"div"> {
  value?: string
  defaultValue?: string
  onValueChange?: (value: string) => void
}

function Tabs({
  value: controlledValue,
  defaultValue = "",
  onValueChange,
  children,
  className,
  ...props
}: TabsProps) {
  const [internalValue, setInternalValue] = React.useState(defaultValue)
  const value = controlledValue ?? internalValue
  const handleChange = React.useCallback(
    (v: string) => {
      setInternalValue(v)
      onValueChange?.(v)
    },
    [onValueChange],
  )
  return (
    <TabsContext value={{ value, onValueChange: handleChange }}>
      <div className={cn("w-full", className)} {...props}>
        {children}
      </div>
    </TabsContext>
  )
}

interface TabsTriggerProps extends React.ComponentProps<"button"> {
  value: string
}

function TabsList({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      role="tablist"
      className={cn(
        "inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground",
        className,
      )}
      {...props}
    />
  )
}

function TabsTrigger({ className, value, ...props }: TabsTriggerProps) {
  const ctx = React.useContext(TabsContext)
  const isActive = ctx.value === value

  const handleKeyDown = (e: React.KeyboardEvent<HTMLButtonElement>) => {
    const keys = ["ArrowLeft", "ArrowRight", "Home", "End"]
    if (!keys.includes(e.key)) return
    e.preventDefault()
    const list = e.currentTarget.closest('[role="tablist"]')
    if (!list) return
    const tabs = Array.from(list.querySelectorAll<HTMLButtonElement>('[role="tab"]:not(:disabled)'))
    const current = tabs.indexOf(e.currentTarget)
    let next = current
    if (e.key === "ArrowLeft") next = (current - 1 + tabs.length) % tabs.length
    if (e.key === "ArrowRight") next = (current + 1) % tabs.length
    if (e.key === "Home") next = 0
    if (e.key === "End") next = tabs.length - 1
    tabs[next]?.focus()
    tabs[next]?.click()
  }

  return (
    <button
      role="tab"
      id={tabId(value)}
      aria-selected={isActive}
      aria-controls={panelId(value)}
      tabIndex={isActive ? 0 : -1}
      data-state={isActive ? "active" : "inactive"}
      className={cn(
        "inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-sm font-medium ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
        isActive && "bg-background text-foreground shadow",
        className,
      )}
      onClick={() => ctx.onValueChange(value)}
      onKeyDown={handleKeyDown}
      {...props}
    />
  )
}

interface TabsContentProps extends React.ComponentProps<"div"> {
  value: string
}

function TabsContent({ className, value, ...props }: TabsContentProps) {
  const ctx = React.useContext(TabsContext)
  if (ctx.value !== value) return null
  return (
    <div
      role="tabpanel"
      id={panelId(value)}
      aria-labelledby={tabId(value)}
      tabIndex={0}
      className={cn(
        "mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
        className,
      )}
      {...props}
    />
  )
}

export { Tabs, TabsList, TabsTrigger, TabsContent }
