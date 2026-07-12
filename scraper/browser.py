from playwright.sync_api import sync_playwright, Browser, BrowserContext, Page
from playwright_stealth import Stealth
from contextlib import contextmanager

_USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

_browser: Browser | None = None
_playwright = None


def _ensure_browser():
    global _browser, _playwright
    if _browser:
        return
    _playwright = sync_playwright().start()
    _browser = _playwright.chromium.launch(headless=True)


@contextmanager
def get_page():
    _ensure_browser()
    context: BrowserContext = _browser.new_context(
        user_agent=_USER_AGENT,
        viewport={"width": 1280, "height": 720},
    )
    Stealth().apply_stealth_sync(context)
    page: Page = context.new_page()
    try:
        yield page
    finally:
        context.close()


def close_browser():
    global _browser, _playwright
    if _browser:
        _browser.close()
        _browser = None
    if _playwright:
        _playwright.stop()
        _playwright = None
