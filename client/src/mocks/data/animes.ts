import type { Anime, Episode, EpisodeType, Season } from "@/api/types"

function createEpisode(
  animeSlug: string,
  seasonNumber: number,
  episodeNumber: number,
  type: EpisodeType,
): Episode {
  return {
    id: `${animeSlug}-s${seasonNumber}-e${episodeNumber}`,
    title: `Episódio ${episodeNumber}`,
    number: String(episodeNumber),
    seasonNumber,
    type,
    url: `https://animesonlinecc.to/episodio/${animeSlug}-${seasonNumber}x${episodeNumber}`,
  }
}

function createSeason(
  animeSlug: string,
  seasonNumber: number,
  episodeCount: number,
  type: EpisodeType = "episode",
  label?: string,
): Season {
  return {
    number: seasonNumber,
    label: label ?? `Temporada ${seasonNumber}`,
    type,
    episodes: Array.from({ length: episodeCount }, (_, i) =>
      createEpisode(animeSlug, seasonNumber, i + 1, type),
    ),
  }
}

function createOva(animeSlug: string, ovaId: string, title: string, number: number): Episode {
  return {
    id: ovaId,
    title,
    number: String(number),
    seasonNumber: 0,
    type: "ova",
    url: `https://animesonlinecc.to/episodio/${animeSlug}-ova-${number}`,
  }
}

export const mockAnimes: Anime[] = [
  {
    id: "shingeki-no-kyojin",
    title: "Shingeki no Kyojin",
    url: "https://animesonlinecc.to/anime/shingeki-no-kyojin",
    imageUrl: "https://placehold.co/300x450/1a1a2e/f472b6?text=SnK",
    description:
      "Num mundo onde a humanidade vive cercada por muralhas gigantes para se proteger dos Titãs...",
    genres: ["Ação", "Drama"],
    seasons: [
      createSeason("shingeki-no-kyojin", 1, 25),
      createSeason("shingeki-no-kyojin", 2, 12),
      createSeason("shingeki-no-kyojin", 3, 22),
      createSeason("shingeki-no-kyojin", 4, 28),
      {
        number: 0,
        label: "OVAs",
        type: "ova",
        episodes: [
          createOva("shingeki-no-kyojin", "shingeki-ova-1", "OVA 1", 1),
          createOva("shingeki-no-kyojin", "shingeki-ova-2", "OVA 2", 2),
        ],
      },
    ],
  },
  {
    id: "one-punch-man",
    title: "One Punch Man",
    url: "https://animesonlinecc.to/anime/one-punch-man",
    imageUrl: "https://placehold.co/300x450/1a1a2e/f472b6?text=OPM",
    description:
      "Saitama é um herói que consegue derrotar qualquer inimigo com um único soco...",
    genres: ["Ação", "Comédia"],
    seasons: [
      createSeason("one-punch-man", 1, 12),
      createSeason("one-punch-man", 2, 12),
      {
        number: 0,
        label: "Filmes",
        type: "movie",
        episodes: [
          {
            id: "one-punch-man-movie-1",
            title: "One Punch Man: A Hero Nobody Knows",
            number: "1",
            seasonNumber: 0,
            type: "movie",
            url: "https://animesonlinecc.to/episodio/one-punch-man-movie-1",
          },
        ],
      },
    ],
  },
]

export function findAnimeByUrl(url: string): Anime | undefined {
  return mockAnimes.find((anime) => url.includes(anime.id))
}

export function createCatalogAnimes(): Anime[] {
  return mockAnimes.map((anime, animeIndex) => ({
    ...anime,
    seasons: anime.seasons.map((season, seasonIndex) => ({
      ...season,
      episodes: season.episodes.map((episode, episodeIndex) => {
        const baseKey = `animes/${anime.id}/${episode.id}.mp4`
        const hasOriginal = animeIndex === 0 ? episodeIndex < 3 : animeIndex === 1 && seasonIndex === 0 && episodeIndex < 2
        const hasUpscaled = animeIndex === 0 && seasonIndex === 0 && episodeIndex === 0
        return {
          ...episode,
          storageKey: hasOriginal ? baseKey : undefined,
          upscaledStorageKey: hasUpscaled ? baseKey.replace(".mp4", "_upscaled.mp4") : undefined,
        }
      }),
    })),
  }))
}
