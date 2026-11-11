# Ticket Timeline System

A web app where the team logs time spent on tickets and follows each ticket
through its phases, and where managers see everyone's progress and get a
weekly report.

- [Features](#features)
- [Tech stack](#tech-stack)
- [Project structure](#project-structure)
- [Getting started](#getting-started)
- [Configuration reference](#configuration-reference)

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

| Part     | Technology                                                        |
| -------- | ----------------------------------------------------------------- |
| Backend  | Go, using `net/http` with gorilla/mux and `database/sql` with pgx |
| Database | PostgreSQL 17, running in Docker                                  |
| Frontend | Next.js 16 (Pages Router), React 19, TypeScript, CSS Modules      |

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

### How the parts connect

```
Browser ──► Frontend (Next.js, port 3001) ──/api/*──► Backend (Go, port 8000) ──► PostgreSQL (Docker, port 5432)
```

People only ever open the frontend. It forwards every `/api/*` request to
the backend on the same machine, so only port **3001** needs to be reachable
from the network. The backend and database stay internal.

## Getting started

You need Go 1.26+, Node.js 20.9+ with Yarn, and Docker
(for example [Rancher Desktop](https://rancherdesktop.io/) on a Mac).

1. Create `.env` in the project root with `POSTGRES_PASSWORD=...`, then
   start the database:

   ```bash
   docker compose up -d
   ```

2. Start the backend, using the same password:

   ```bash
   export DATABASE_URL="postgres://ticket_timeline:YOUR_PASSWORD@localhost:5432/ticket_timeline?sslmode=disable"
   go run ./cmd/server
   ```

3. In another terminal, start the frontend:

   ```bash
   cd frontend
   yarn install
   yarn dev
   ```

4. Open http://localhost:3001.

Checks to run before pushing:

```bash
go test ./...                        # backend tests
cd frontend && npx tsc --noEmit      # frontend type check
```

## Configuration reference

| Variable            | Used by        | Default                                 | Purpose                                                    |
| ------------------- | -------------- | --------------------------------------- | ---------------------------------------------------------- |
| `POSTGRES_PASSWORD` | Docker         | none (required)                         | Database password                                          |
| `DATABASE_URL`      | backend        | `postgres://postgres:postgres@localhost:5432/ticket_timeline?sslmode=disable` | Database connection |
| `PORT`              | backend        | `8000`                                  | Port the backend listens on                                |
| `COOKIE_SECURE`     | backend        | off                                     | Set to `true` when the site is served over HTTPS           |
| `BACKEND_URL`       | frontend build | `http://localhost:8000`                 | Where the frontend forwards `/api/*`. Set it before `yarn build`. |
