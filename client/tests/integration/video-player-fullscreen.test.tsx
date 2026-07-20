import { describe, expect, it, vi } from "vitest"
import { render, act } from "@testing-library/react"
import { VideoPlayer, enterNativeFullscreen } from "../../src/components/catalog/video-player"

function makeRequestEvent() {
  return new Event("media-enter-fullscreen-request", { bubbles: true, cancelable: true })
}

describe("enterNativeFullscreen", () => {
  it("calls webkitEnterFullscreen and stops the event on iOS video", () => {
    const enter = vi.fn()
    const video = Object.assign(document.createElement("video"), { webkitEnterFullscreen: enter })
    const event = makeRequestEvent()

    const handled = enterNativeFullscreen(video, event)

    expect(handled).toBe(true)
    expect(enter).toHaveBeenCalledOnce()
    expect(event.defaultPrevented).toBe(true)
  })

  it("does nothing when the video lacks native fullscreen", () => {
    const video = document.createElement("video")
    const event = makeRequestEvent()

    const handled = enterNativeFullscreen(video, event)

    expect(handled).toBe(false)
    expect(event.defaultPrevented).toBe(false)
  })

  it("does nothing when there is no video element", () => {
    const event = makeRequestEvent()
    expect(enterNativeFullscreen(null, event)).toBe(false)
    expect(event.defaultPrevented).toBe(false)
  })
})

describe("VideoPlayer fullscreen wiring", () => {
  it("routes the fullscreen request to the native video on iPhone-like devices", () => {
    const enter = vi.fn()
    const { container } = render(
      <VideoPlayer src="/video.mp4" title="Ep 1" episodeId="ep-1" onClose={() => {}} />,
    )

    const wrapper = container.firstElementChild as HTMLElement
    const provider = wrapper.querySelector("[data-media-provider]") ?? wrapper
    const video = Object.assign(document.createElement("video"), { webkitEnterFullscreen: enter })
    provider.appendChild(video)

    const player = wrapper.querySelector("[data-media-player]")
    const event = makeRequestEvent()
    act(() => {
      player!.dispatchEvent(event)
    })

    expect(enter).toHaveBeenCalledOnce()
    expect(event.defaultPrevented).toBe(true)
  })

  it("leaves the request untouched when native fullscreen is unavailable", () => {
    const { container } = render(
      <VideoPlayer src="/video.mp4" title="Ep 1" episodeId="ep-1" onClose={() => {}} />,
    )

    const wrapper = container.firstElementChild as HTMLElement
    const player = wrapper.querySelector("[data-media-player]")
    const event = makeRequestEvent()
    act(() => {
      player!.dispatchEvent(event)
    })

    expect(event.defaultPrevented).toBe(false)
  })
})
