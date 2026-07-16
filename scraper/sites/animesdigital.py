import re
from urllib.parse import urlparse, parse_qs
from browser import get_page

BASE_URL = "https://animesdigital.org"


def resolve_embed(embed_url: str) -> str:
    if "videohls.php" not in embed_url:
        return embed_url
    query = parse_qs(urlparse(embed_url).query)
    direct = query.get("d", [""])[0]
    return direct or embed_url


class AnimesdigitalScraper:
    def scrape_anime(self, url: str) -> dict:
        with get_page() as page:
            page.goto(url, wait_until="networkidle")

            title_el = page.query_selector("h1")
            title = title_el.inner_text().strip() if title_el else ""

            img_el = page.query_selector("img[alt*='Assistir']")
            image_url = img_el.get_attribute("src") if img_el else ""

            desc_el = page.query_selector(".sinopse")
            description = desc_el.inner_text().strip() if desc_el else ""

            ep_links = page.query_selector_all("a.b_flex[href*='/video/']")

            episodes = []
            seen_urls = set()

            for link in ep_links:
                ep_url = link.get_attribute("href") or ""
                if ep_url in seen_urls:
                    continue
                seen_urls.add(ep_url)

                title_el = link.query_selector(".title_anime")
                ep_title = title_el.inner_text().strip() if title_el else ""

                ep_num = ""
                match = re.search(r"Epis[oó]dio\s*(\d+)", ep_title)
                if match:
                    ep_num = match.group(1)

                episodes.append({
                    "title": ep_title,
                    "url": ep_url,
                    "season": "1",
                    "number": ep_num,
                })

            episodes.reverse()

            return {
                "title": title,
                "url": url,
                "imageUrl": image_url or "",
                "description": description,
                "episodes": episodes,
            }

    def scrape_episode(self, url: str) -> list[dict]:
        with get_page() as page:
            page.goto(url, wait_until="networkidle")

            sources = []

            tabs = page.query_selector_all("ul.tabs_videos li")
            tab_labels = [t.inner_text().strip() for t in tabs]

            player_divs = page.query_selector_all("div.tab-video")

            for i, div in enumerate(player_divs):
                iframe = div.query_selector("iframe.metaframe")
                if not iframe:
                    continue
                embed_url = iframe.get_attribute("src") or ""
                if not embed_url:
                    continue
                label = tab_labels[i] if i < len(tab_labels) else f"Player {i + 1}"
                sources.append({"label": label, "embed_url": resolve_embed(embed_url)})

            return sources
