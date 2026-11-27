import type { Metadata } from "next";
import Link from "next/link";
import { CodeReviewIcon, GitMergeIcon } from "@primer/octicons-react";
import { api } from "@/lib/api";
import type { Repository } from "@/lib/types";
import { reviewsAcross } from "@/lib/reviews";
import { ReviewRow } from "@/components/ReviewRow";
import { Blankslate } from "@/components/Blankslate";

export const metadata: Metadata = { title: "Reviews" };

export default async function ReviewsPage({ searchParams }: { searchParams: Promise<{ state?: string }> }) {
  const { state = "OPEN" } = await searchParams;
  const st = ["OPEN", "LANDED", "CLOSED"].includes(state) ? state : "OPEN";
  const { repositories } = await api<{ repositories: Repository[] }>("/v1/repositories?pageSize=100");
  const reviews = await reviewsAcross(repositories, st);
  return (
    <div className="gh-container">
      <h1 className="f3 text-normal mb-3">Reviews</h1>
      <div className="Box">
        <div className="Box-header d-flex flex-items-center" style={{ gap: 16 }}>
          <Link href="/reviews?state=OPEN" className={st === "OPEN" ? "color-fg-default text-bold" : "color-fg-muted"}>
            <CodeReviewIcon /> Open
          </Link>
          <Link href="/reviews?state=LANDED" className={st === "LANDED" ? "color-fg-default text-bold" : "color-fg-muted"}>
            <GitMergeIcon /> Landed
          </Link>
          <Link href="/reviews?state=CLOSED" className={st === "CLOSED" ? "color-fg-default text-bold" : "color-fg-muted"}>
            Closed
          </Link>
        </div>
        {reviews.length === 0 ? (
          <Blankslate icon={<CodeReviewIcon size={24} />} title="No reviews here" />
        ) : (
          <ul className="list-style-none">
            {reviews.map((rv) => (
              <ReviewRow key={rv.name} repo={rv.repositoryId} review={rv} showRepo />
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
