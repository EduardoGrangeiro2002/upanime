import re
from browser import get_page

BASE_URL = "https://akumanimes.com"

_EPISODE_NUM_RE = re.compile(r"Epis[oó]dio\s*(\d+)", re.IGNORECASE)
_SINOPSE_PREFIX_RE = re.compile(r"^Sinopse:\s*", re.IGNORECASE)


def parse_episode_number(title: str) -> str:
    match = _EPISODE_NUM_RE.search(title)
    if not match:
        return ""
    return match.group(1)


class AkumanimesScraper:
    def scrape_anime(self, url: str) -> dict:
        with get_page() as page:
            page.goto(url, wait_until="domcontentloaded")
            page.wait_for_selector(".ak2-list-block", timeout=15000)

            title_el = page.query_selector("h1")
            title = title_el.inner_text().strip() if title_el else ""

            img_el = page.query_selector("#capaAnime img")
            image_url = img_el.get_attribute("src") if img_el else ""

            desc_el = page.query_selector("#sinopse2")
            description = desc_el.inner_text().strip() if desc_el else ""
            description = _SINOPSE_PREFIX_RE.sub("", description)

            episodes = []
            seen_urls = set()
            links = page.query_selector_all(".ak2-list-block a")

            for link in links:
                ep_url = link.get_attribute("href") or ""
                if not ep_url or ep_url in seen_urls:
                    continue
                item = link.query_selector("li.rl_CLitem")
                if not item:
                    continue
                seen_urls.add(ep_url)
                ep_title = item.inner_text().strip()
                episodes.append({
                    "title": ep_title,
                    "url": ep_url,
                    "season": "1",
                    "number": parse_episode_number(ep_title),
                })

            return {
                "title": title,
                "url": url,
                "imageUrl": image_url or "",
                "description": description,
                "episodes": episodes,
            }

    def scrape_episode(self, url: str) -> list[dict]:
        with get_page() as page:
            page.goto(url, wait_until="domcontentloaded")
            page.wait_for_selector(".riverlab-player-surface[data-iframe-src]", timeout=15000)

            sources = []
            surfaces = page.query_selector_all(".riverlab-player-surface[data-iframe-src]")

            for surface in surfaces:
                embed_url = surface.get_attribute("data-iframe-src") or ""
                if not embed_url:
                    continue
                label = (surface.get_attribute("data-player-label") or "").strip() or "Fonte"
                sources.append({"label": label, "embed_url": embed_url})

            return sources
