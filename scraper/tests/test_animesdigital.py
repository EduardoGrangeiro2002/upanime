import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from sites import get_scraper
from sites.animesdigital import AnimesdigitalScraper, resolve_embed


def test_get_scraper_returns_animesdigital():
    assert isinstance(get_scraper("animesdigital.org"), AnimesdigitalScraper)


def test_resolve_embed_extracts_m3u8_from_videohls():
    embed = "https://api.anivideo.net/videohls.php?d=https://cdn.imagesskill.com/stream/g/golden-time/01.mp4/index.m3u8&nocache1783996198"
    assert resolve_embed(embed) == "https://cdn.imagesskill.com/stream/g/golden-time/01.mp4/index.m3u8"


def test_resolve_embed_keeps_videohls_without_d_param():
    embed = "https://api.anivideo.net/videohls.php?nocache123"
    assert resolve_embed(embed) == embed


def test_resolve_embed_keeps_other_urls():
    embed = "https://www.blogger.com/video.g?token=abc"
    assert resolve_embed(embed) == embed
