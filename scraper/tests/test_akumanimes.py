import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from sites import get_scraper
from sites.akumanimes import AkumanimesScraper, parse_episode_number


def test_get_scraper_returns_akumanimes():
    assert isinstance(get_scraper("akumanimes.com"), AkumanimesScraper)


def test_parse_episode_number():
    assert parse_episode_number("Dragon Ball (Dublado) – Dublado – Episódio 01 – O Segredo das Esferas do Dragão") == "01"
    assert parse_episode_number("Dragon Ball – Episódio 153 – Final") == "153"
    assert parse_episode_number("Dragon Ball – Filme 1") == ""
    assert parse_episode_number("") == ""
