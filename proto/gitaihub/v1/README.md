# GitaiHub API contract (v1)

These protobuf files are the written contract of GitaiHub's HTTP/JSON API,
following the [Google Cloud API design guide](https://cloud.google.com/apis/design):

- resources have names like `repositories/orbit/reviews/1`;
- collections use standard `List` methods with `page_size` / `page_token` /
  `next_page_token`, and single resources use `Get`;
- the few non-standard actions are custom methods (`:submitVerdict`, `:watch`);
- errors use the Google error model (`{"error": {"code", "message", "status"}}`,
  `status` being the canonical code name such as `NOT_FOUND`).

The Go server in `api/internal/httpapi` implements this contract by hand with
the proto3 JSON mapping (lowerCamelCase fields, int64 as strings, RFC 3339
timestamps). The files are not compiled yet: they document the API and are
the starting point for generated clients or a gRPC gateway later.

Services map to the bounded contexts:

| Service | Context | Upstream in gitai |
| --- | --- | --- |
| `SourceService` | Source | RefService, CommitService, DiffService, DerivedDataService (blame), SearchService |
| `ReviewService` | Review | ReviewService |
| `ActivityService` | Agent activity | ChangesetService (copied into Postgres) |
| `IdentityService` | Identity | none (GitaiHub's own users) |
