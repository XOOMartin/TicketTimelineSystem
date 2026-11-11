# Ticket Timeline System

A web app where the team logs time spent on tickets and follows each ticket
through its phases, and where managers see everyone's progress and get a
weekly report.

## Features

**For members**

- **My timeline**: every ticket you're working on, with its status, start
  and end dates, and the hours spent in each phase.
- **Timer**: start a timer on a ticket and the time is logged when you stop.
  Time under 1 minute isn't recorded. If you close the site while a timer is
  running, it stops and saves the time up to when you left.
- **Progress form**: move a ticket step by step through its phases:
  - Spec (Check Spec / Investigation)
  - Confirm (Confirmation / Review to SE/MGRs)
  - Planning
  - Coding
  - Testing
  - Review (by a manager)

  When a ticket comes back for changes, a new round starts (Revision 1, 2…).
- **Check-ins**: add, edit and delete your logged time.
- **Help**: explains every term the app uses.

**For admins**

- **Dashboard**: who is behind on check-ins, and which tickets are blocked
  or waiting.
- **Members**: each person's week, and importing a member's existing
  progress spreadsheet (.xlsx).
- **Reports**: generated automatically every Friday at 17:00 or on demand,
  and downloadable as Excel. You can also download a team sheet.
- **Users**: create, edit, delete and restore accounts.

The first account to sign up becomes an admin.

## Tech stack

| Part     | Technology                                                       |
| -------- | ---------------------------------------------------------------- |
| Backend  | Go, using `net/http` with gorilla/mux and `database/sql` with pgx |
| Database | PostgreSQL 17, running in Docker                                 |
| Frontend | Next.js 16 (Pages Router), React 19, TypeScript, CSS Modules     |

## Project structure

```
cmd/server/              Entry point: wires everything together and starts the API
internal/
  domain/                Business rules: tickets, phases, statuses, timers, reports, users
  application/           Use cases (services) that coordinate the domain and repositories
  infrastructure/        PostgreSQL repositories, password hashing
  interfaces/
    rest/                HTTP handlers and JSON request/response types
    excel/               Spreadsheet import and export
frontend/
  pages/                 One file per page (timeline, progress, check-ins, admin…)
  components/            UI components: atoms → molecules → organisms
  hooks/                 Data loading and state logic used by the pages
  lib/                   API client, labels, glossary, helpers
  styles/                CSS Modules
docker-compose.yml       The PostgreSQL database
```

## Getting started

### Requirements

- [Go](https://go.dev/dl/) 1.26 or later
- [Node.js](https://nodejs.org/) 20.9 or later, and [Yarn](https://yarnpkg.com/)
- Docker, for example through
  [Rancher Desktop](https://rancherdesktop.io/) or Docker Desktop

### 1. Start the database

Create a file called `.env` in the project root with a password of your
choice:

```
POSTGRES_PASSWORD=choose-a-password
```

`.env` is in `.gitignore`, so it is never committed.

Then start PostgreSQL:

```bash
docker compose up -d
```

The database is only reachable from the same machine (`127.0.0.1:5432`).
Its data is kept in a Docker volume, so it survives restarts.

### 2. Start the backend

The server reads its database address from the `DATABASE_URL` environment
variable. Use the same password as in `.env`:

```bash
export DATABASE_URL="postgres://ticket_timeline:choose-a-password@localhost:5432/ticket_timeline?sslmode=disable"
go run ./cmd/server
```

The API runs on http://localhost:8000. The database tables are created
automatically on first start.

### 3. Start the frontend

In another terminal:

```bash
cd frontend
yarn install
yarn dev
```

Open http://localhost:3001 and sign up. The first account becomes the
admin.

The frontend forwards every `/api/*` request to the Go server, so the
browser only ever talks to one address.

## Configuration

| Variable       | Used by  | Default                                                                | Purpose                                                       |
| -------------- | -------- | ---------------------------------------------------------------------- | ------------------------------------------------------------- |
| `DATABASE_URL` | backend  | `postgres://postgres:postgres@localhost:5432/ticket_timeline?sslmode=disable` | PostgreSQL connection                                 |
| `PORT`         | backend  | `8000`                                                                 | Port the API listens on                                       |
| `COOKIE_SECURE`| backend  | off                                                                    | Set to `true` when the site is served over HTTPS              |
| `BACKEND_URL`  | frontend | `http://localhost:8000`                                                | Where the frontend forwards `/api/*` requests                 |
| `POSTGRES_PASSWORD` | docker compose | none (required)                                             | Database password, set in `.env`                              |

## Running on a server

```bash
# Database
docker compose up -d

# Backend: build once, then run the binary
go build -o server ./cmd/server
DATABASE_URL="postgres://ticket_timeline:<password>@localhost:5432/ticket_timeline?sslmode=disable" ./server

# Frontend: build once, then serve
cd frontend
yarn install
yarn build
yarn start          # serves on port 3001
```

Keep both processes running, for example with systemd or another process
manager. If the site is served over HTTPS, set `COOKIE_SECURE=true` for the
backend.

### Moving existing data to another machine

Back up from the old machine and restore on the new one with PostgreSQL's
own tools, which are included in the database container:

```bash
# On the old machine
docker compose exec db pg_dump -U ticket_timeline -Fc ticket_timeline > backup.dump

# On the new machine, after `docker compose up -d`
docker compose exec -T db pg_restore -U ticket_timeline -d ticket_timeline --clean --if-exists < backup.dump
```

## Development

```bash
go test ./...                        # backend tests
cd frontend && npx tsc --noEmit      # frontend type check
```
