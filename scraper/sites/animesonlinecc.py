import re
from browser import get_page

BASE_URL = "https://animesonlinecc.to"


class AnimesonlineccScraper:
    def scrape_anime(self, url: str) -> dict:
        with get_page() as page:
            page.goto(url, wait_until="networkidle")

            title_el = page.query_selector(".sheader .data h1")
            title = title_el.inner_text() if title_el else ""

            img_el = page.query_selector(".sheader .poster img")
            image_url = img_el.get_attribute("src") if img_el else ""

            desc_el = page.query_selector("#info .wp-content p")
            description = desc_el.inner_text() if desc_el else ""

            season_headers = page.query_selector_all(".se-q")
            for header in season_headers:
                header.click()
                page.wait_for_timeout(300)

            episodes = []
            seen_urls = set()
            season_containers = page.query_selector_all(".se-c")

            for season_idx, container in enumerate(season_containers, start=1):
                all_headers = page.query_selector_all(".se-q")
                season_num = str(season_idx)
                if season_idx <= len(all_headers):
                    season_title_el = all_headers[season_idx - 1].query_selector(".se-t")
                    if season_title_el:
                        season_text = season_title_el.inner_text()
                        match = re.search(r"\d+", season_text)
                        if match:
                            season_num = match.group(0)

                links = container.query_selector_all("li a[href*='/episodio/']")
                for link in links:
                    ep_url = link.get_attribute("href") or ""
                    if ep_url in seen_urls:
                        continue
                    seen_urls.add(ep_url)
                    ep_title_el = link.query_selector(".episodiotitle .epst") or link.query_selector(".episodiotitle")
                    ep_num_el = link.query_selector(".episodiotitle .numerando")
                    ep_title = ep_title_el.inner_text() if ep_title_el else ep_url
                    ep_num = ""
                    if ep_num_el:
                        num_text = ep_num_el.inner_text()
                        match = re.search(r"(\d+)$", num_text)
                        if match:
                            ep_num = match.group(1)
                    episodes.append({
                        "title": ep_title.strip(),
                        "url": ep_url,
                        "season": season_num,
                        "number": ep_num,
                    })

            return {
                "title": title.strip(),
                "url": url,
                "imageUrl": image_url or "",
                "description": description.strip(),
                "episodes": episodes,
            }

    def scrape_episode(self, url: str) -> list[dict]:
        with get_page() as page:
            page.goto(url, wait_until="networkidle")
            tabs = page.query_selector_all("ul.idTabs.sourceslist li a")
            boxes = page.query_selector_all("div.play-box-iframe")
            sources = []
            for i, box in enumerate(boxes):
                iframe = box.query_selector("iframe.metaframe")
                if not iframe:
                    continue
                embed_url = iframe.get_attribute("src") or ""
                if not embed_url:
                    continue
                label = "Fonte"
                if i < len(tabs):
                    label = tabs[i].inner_text().strip()
                sources.append({"label": label, "embed_url": embed_url})
            return sources
