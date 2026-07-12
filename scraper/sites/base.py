from typing import Protocol, TypedDict


class EpisodeEntry(TypedDict):
    title: str
    url: str
    season: str
    number: str


class AnimeResult(TypedDict):
    title: str
    url: str
    imageUrl: str
    description: str
    episodes: list[EpisodeEntry]


class EpisodeSource(TypedDict):
    label: str
    embed_url: str


class SiteScraper(Protocol):
    def scrape_anime(self, url: str) -> AnimeResult: ...
    def scrape_episode(self, url: str) -> list[EpisodeSource]: ...
