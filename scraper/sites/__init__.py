from sites.animesonlinecc import AnimesonlineccScraper
from sites.animesdigital import AnimesdigitalScraper
from sites.akumanimes import AkumanimesScraper

SCRAPERS = {
    "animesonlinecc.to": AnimesonlineccScraper(),
    "animesdigital.org": AnimesdigitalScraper(),
    "akumanimes.com": AkumanimesScraper(),
}


def get_scraper(domain: str):
    return SCRAPERS.get(domain)
