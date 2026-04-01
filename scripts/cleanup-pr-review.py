#!/usr/bin/env python3
"""
Clean up outdated bot review noise from a GitHub pull request.

The latest bot review is kept; older bot reviews are resolved/minimized.
Progress tracking comments are deleted; other outdated bot issue comments
are minimized as outdated.

Only targets reviews/comments by claude[bot] and github-actions[bot].

Usage: uv run scripts/cleanup-pr-review.py <owner> <repo> <pr_number>
Requires: gh (authenticated)
"""

import json
import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor

# Bot logins we manage — other bots (e.g. gemini) are left untouched.
# REST API returns logins with [bot] suffix; GraphQL returns them without.
BOT_LOGINS_REST = {"claude[bot]", "github-actions[bot]"}
BOT_LOGINS_GQL = {"claude", "github-actions"}

MINIMIZE_MUTATION = """
mutation($id: ID!) {
  minimizeComment(input: {subjectId: $id, classifier: OUTDATED}) {
    minimizedComment { isMinimized }
  }
}
"""

RESOLVE_THREAD_MUTATION = """
mutation($threadId: ID!) {
  resolveReviewThread(input: {threadId: $threadId}) {
    thread { isResolved }
  }
}
"""

REVIEW_THREADS_QUERY = """
query($owner: String!, $repo: String!, $pr: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $pr) {
      reviewThreads(last: 100) {
        nodes {
          id
          isResolved
          comments(first: 100) {
            nodes {
              author { login }
              databaseId
            }
          }
        }
      }
    }
  }
}
"""


def gh(*args: str) -> str:
    result = subprocess.run(
        ["gh", *args],
        capture_output=True,
        text=True,
        check=True,
    )
    return result.stdout


def gh_api(endpoint: str, *, paginate: bool = False) -> list | dict:
    args = ["api", endpoint]
    if paginate:
        # --slurp wraps each page in an outer array so json.loads can parse
        # concatenated multi-page output (without it, [page1][page2] is invalid JSON).
        args.extend(["--paginate", "--slurp"])
    result = json.loads(gh(*args))
    # --slurp produces [[...], [...], ...]; flatten into a single list.
    if paginate and isinstance(result, list) and result and isinstance(result[0], list):
        return [item for page in result for item in page]
    return result


def gh_graphql(query: str, **variables: str | int) -> dict:
    args = ["api", "graphql", "-f", f"query={query}"]
    for key, value in variables.items():
        flag = "-F" if isinstance(value, int) else "-f"
        args.extend([flag, f"{key}={value}"])
    return json.loads(gh(*args))


def gh_fire_and_forget(*args: str) -> None:
    """Run a gh command. Errors are logged but not raised."""
    result = subprocess.run(["gh", *args], capture_output=True, text=True, check=False)
    if result.returncode != 0:
        print(f"[warn] gh {' '.join(args[:3])}... failed (exit {result.returncode}): {result.stderr.strip()}", file=sys.stderr)


def is_bot_rest(login: str) -> bool:
    return login in BOT_LOGINS_REST


def is_bot_gql(login: str | None) -> bool:
    # GitHub returns author: null for deleted/ghost accounts.
    return login is not None and login in BOT_LOGINS_GQL


def classify_outdated_threads(
    threads: list[dict],
    outdated_comment_ids: set[int],
    comment_id_to_review_id: dict[int, int],
) -> tuple[list[str], set[int]]:
    """Single pass over threads: return (thread_ids_to_resolve, protected_review_ids).

    Threads whose first comment belongs to an outdated bot review are classified:
    - All-bot threads → resolve them.
    - Threads with human comments → protect their parent review from minimization.
    """
    to_resolve: list[str] = []
    protected: set[int] = set()

    for thread in threads:
        if thread["isResolved"]:
            continue
        comments = thread["comments"]["nodes"]
        if not comments:
            continue
        first_db_id = comments[0]["databaseId"]
        if first_db_id not in outdated_comment_ids:
            continue

        all_bots = all(
            is_bot_gql(c["author"]["login"] if c.get("author") else None)
            for c in comments
        )
        if all_bots:
            to_resolve.append(thread["id"])
        else:
            review_id = comment_id_to_review_id.get(first_db_id)
            if review_id is not None:
                protected.add(review_id)

    return to_resolve, protected


def resolve_threads(thread_ids: list[str]) -> int:
    """Resolve the given review threads concurrently."""
    with ThreadPoolExecutor(max_workers=5) as pool:
        for tid in thread_ids:
            pool.submit(
                gh_fire_and_forget,
                "api", "graphql",
                "-f", f"query={RESOLVE_THREAD_MUTATION}",
                "-f", f"threadId={tid}",
            )
    return len(thread_ids)


def minimize_outdated_reviews(
    bot_reviews: list[dict],
    protected_review_ids: set[int],
) -> int:
    """Minimize outdated bot review summaries, skipping protected ones."""
    outdated = [
        r for r in bot_reviews[1:]
        if r.get("body")
        and r["id"] not in protected_review_ids
    ]

    with ThreadPoolExecutor(max_workers=5) as pool:
        for review in outdated:
            pool.submit(
                gh_fire_and_forget,
                "api", "graphql",
                "-f", f"query={MINIMIZE_MUTATION}",
                "-f", f"id={review['node_id']}",
            )

    return len(outdated)


def cleanup_issue_comments(owner: str, repo: str, pr_number: int) -> None:
    """Delete progress tracking comments and minimize outdated bot issue comments."""
    print("Fetching issue comments...")
    issue_comments: list[dict] = gh_api(
        f"repos/{owner}/{repo}/issues/{pr_number}/comments", paginate=True
    )

    bot_comments = [c for c in issue_comments if is_bot_rest(c["user"]["login"])]

    # Partition into progress comments and substantive comments.
    # These patterns match comments created by anthropics/claude-code-action
    # with track_progress: true. Update if that action changes its format.
    progress_comments: list[dict] = []
    substantive_comments: list[dict] = []
    for c in bot_comments:
        body = c.get("body", "")
        is_progress = (
            body.startswith("**Claude finished @")
            or "Claude Code is working" in body
            or body.startswith("### Code Review:")
        )
        if is_progress:
            progress_comments.append(c)
        else:
            substantive_comments.append(c)

    # Delete progress tracking comments (ephemeral, no value keeping them)
    with ThreadPoolExecutor(max_workers=5) as pool:
        for c in progress_comments:
            pool.submit(
                gh_fire_and_forget,
                "api", "-X", "DELETE",
                f"repos/{owner}/{repo}/issues/comments/{c['id']}",
            )
    print(f"Deleted {len(progress_comments)} progress comment(s).")

    # Minimize outdated bot issue comments (keep the most recent one)
    sorted_substantive = sorted(
        substantive_comments, key=lambda c: c["created_at"], reverse=True
    )
    outdated = sorted_substantive[1:]

    with ThreadPoolExecutor(max_workers=5) as pool:
        for c in outdated:
            pool.submit(
                gh_fire_and_forget,
                "api", "graphql",
                "-f", f"query={MINIMIZE_MUTATION}",
                "-f", f"id={c['node_id']}",
            )
    print(f"Minimized {len(outdated)} outdated issue comment(s).")


def main() -> None:
    if len(sys.argv) != 4:
        print(f"Usage: {sys.argv[0]} <owner> <repo> <pr_number>", file=sys.stderr)
        sys.exit(1)

    owner = sys.argv[1]
    repo = sys.argv[2]
    pr_number = int(sys.argv[3])

    # --- Step 1: Fetch all bot output ---
    print("Fetching inline comments and reviews...")
    comments: list[dict] = gh_api(
        f"repos/{owner}/{repo}/pulls/{pr_number}/comments", paginate=True
    )
    reviews: list[dict] = gh_api(
        f"repos/{owner}/{repo}/pulls/{pr_number}/reviews", paginate=True
    )

    # --- Step 2: Identify the latest bot review ---
    bot_reviews = sorted(
        [r for r in reviews if is_bot_rest(r["user"]["login"])],
        key=lambda r: r["submitted_at"],
        reverse=True,
    )

    print(f"Found {len(bot_reviews)} bot review(s).")

    if len(bot_reviews) > 1:
        latest_review_id = bot_reviews[0]["id"]
        print(f"Latest bot review ID: {latest_review_id}")

        # Comments belonging to non-latest bot reviews
        outdated_comment_ids: set[int] = {
            c["id"]
            for c in comments
            if is_bot_rest(c["user"]["login"])
            and c["pull_request_review_id"] != latest_review_id
        }

        # --- Steps 3-4: Classify threads and clean up ---
        print("Fetching review threads...")
        threads_response = gh_graphql(
            REVIEW_THREADS_QUERY,
            owner=owner,
            repo=repo,
            pr=pr_number,
        )
        threads = (
            threads_response["data"]["repository"]["pullRequest"]
            ["reviewThreads"]["nodes"]
        )

        # Build mapping: comment databaseId -> pull_request_review_id
        comment_id_to_review_id = {
            c["id"]: c["pull_request_review_id"] for c in comments
        }

        thread_ids_to_resolve, protected = classify_outdated_threads(
            threads, outdated_comment_ids, comment_id_to_review_id
        )

        resolved = resolve_threads(thread_ids_to_resolve)
        print(f"Resolved {resolved} outdated thread(s).")

        minimized = minimize_outdated_reviews(bot_reviews, protected)
        print(f"Minimized {minimized} outdated review summary(ies).")
    else:
        print("Skipping thread resolution and review minimization (need >1 bot review).")

    # --- Step 5: Clean up bot issue comments ---
    cleanup_issue_comments(owner, repo, pr_number)

    print("Cleanup complete.")


if __name__ == "__main__":
    main()
