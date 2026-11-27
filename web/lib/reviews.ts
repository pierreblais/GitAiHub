import "server-only";
import { api, enc } from "./api";
import type { Repository, Review } from "./types";

export interface RepoReview extends Review {
  repositoryId: string;
}

/** Reviews in a given state across every repository, most recently updated first. */
export async function reviewsAcross(repositories: Repository[], state = "OPEN"): Promise<RepoReview[]> {
  const lists = await Promise.all(
    repositories.map((r) =>
      api<{ reviews: Review[] }>(`/v1/repositories/${enc(r.repositoryId)}/reviews?state=${state}`)
        .then(({ reviews }) => reviews.map((rv) => ({ ...rv, repositoryId: r.repositoryId })))
        .catch(() => [] as RepoReview[]),
    ),
  );
  return lists.flat().sort((a, b) => (b.updateTime ?? "").localeCompare(a.updateTime ?? ""));
}
