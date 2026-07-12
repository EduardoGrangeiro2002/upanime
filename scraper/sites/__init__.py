from sites.animesonlinecc import AnimesonlineccScraper
from sites.animesdigital import AnimesdigitalScraper

SCRAPERS = {
    "animesonlinecc.to": AnimesonlineccScraper(),
    "animesdigital.org": AnimesdigitalScraper(),
}


def get_scraper(domain: str):
    return SCRAPERS.get(domain)
