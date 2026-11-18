# Delcom Blog — Backend

REST API Go + Gin, GORM, PostgreSQL 17, JWT, role-based admin, article Markdown CRUD and Search Engine-grounded AI article drafting.

## Requirements
- Go 1.27.4 (module target)
- PostgreSQL 17

## Setup
1. Create database: `CREATE DATABASE pabwe_p9_blog;`
2. Copy `.env.example` to `.env`, set database credentials, a long `APP_KEY`, and valid `OPENAI_KEY` / `OPENAI_MODEL`.
3. Run `go mod download`
4. Run `go run ./cmd/server`
5. API: `http://localhost:8081`; Swagger: `http://localhost:8081/swagger/index.html`.

Run backend unit tests with `go test -v -cover ./internal/...`. The repository includes focused article-domain tests; full 100% package coverage still needs to be completed and verified in the Go 1.27.4 environment.

The server migrates `users` and `posts` and seeds the admin from `ADMIN_USERNAME` / `ADMIN_PASSWORD`. Public registration always creates a member; never grant admin via a registration payload.

## Endpoints
- `GET /api/public/categories`, `GET /api/public/posts?q=&category=`, `GET /api/public/posts/:slug`
- `POST /api/auth/register`, `POST /api/auth/login`, `POST /api/auth/logout`, `GET|PUT /api/auth/profile`
- Bearer-authenticated author: `GET|POST /api/posts`, `GET /api/posts/stats`, `GET|PUT|DELETE /api/posts/:id`, `PATCH /api/posts/:id/toggle-publish`, `POST /api/posts/ai-generate`
- Admin: `GET /api/admin/users`, `GET /api/admin/posts`, `GET /api/admin/users/:id/posts`

All API responses use `{status,message,data?}`. Reading time is estimated at 200 words/minute. Slugs are unique. Article content is Markdown.

## AI Search flow
When `include_web_search=true`, the backend queries `SEARCH_ENGINE_ENDPOINT`, passes returned snippets into the configured OpenAI-compatible `/chat/completions` endpoint, and returns the generated article with `search_sources`. Configure provider credentials; no fake/offline AI result is returned when credentials are missing.
