package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"

	"crud_app/internal/application"
	"crud_app/internal/infrastructure/password"
	"crud_app/internal/infrastructure/postgres"
	"crud_app/internal/interfaces/rest"
)

// main is the composition root: it wires infrastructure into the
// application layer and exposes it over HTTP.
func main() {
	db, err := postgres.Open(envOr("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ticket_timeline?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	users := postgres.NewUserRepository(db)
	checkIns := postgres.NewCheckInRepository(db)

	sessions := postgres.NewSessionStore(db)
	hasher := password.NewBcrypt()
	authService := application.NewAuthService(users, sessions, hasher)
	userAdminService := application.NewUserAdminService(users, sessions, hasher)
	ticketDates := postgres.NewTicketDatesRepository(db)
	progress := postgres.NewProgressRepository(db)
	planned := postgres.NewPlannedTicketRepository(db)
	parents := postgres.NewParentRepository(db)
	types := postgres.NewTypeRepository(db)

	checkInService := application.NewCheckInService(checkIns, ticketDates, progress, planned, parents, types)
	memberService := application.NewMemberService(users, checkIns, postgres.NewImportRepository(db), ticketDates, progress, planned, parents, types)
	timerService := application.NewTimerService(postgres.NewTimerRepository(db), checkIns, progress, types, parents, planned)
	reportService := application.NewReportService(postgres.NewReportRepository(db), checkIns, users)

	go runWeeklyReports(reportService)
	go stopClosedTimers(timerService)

	authHandler := rest.NewAuthHandler(authService, os.Getenv("COOKIE_SECURE") == "true")

	r := mux.NewRouter()
	rest.RegisterPublic(r)
	authHandler.Register(r)

	members := r.NewRoute().Subrouter()
	members.Use(authHandler.RequireUser)
	rest.NewCheckInHandler(checkInService).Register(members)
	rest.NewTimerHandler(timerService).Register(members)

	admins := members.NewRoute().Subrouter()
	admins.Use(rest.RequireAdmin)
	rest.NewAdminHandler(memberService, reportService, userAdminService).Register(admins)

	port := envOr("PORT", "8000")
	fmt.Printf("Server is running on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// runWeeklyReports checks every minute whether the Friday report is due.
// Checking often (instead of sleeping until Friday) means a restart or a
// laptop waking from sleep can't make it miss the week.
func runWeeklyReports(reports *application.ReportService) {
	for {
		rp, err := reports.GenerateWeeklyIfDue(context.Background(), time.Now())
		if err != nil {
			log.Printf("weekly report: %v", err)
		} else if rp != nil {
			log.Printf("weekly report generated for %s to %s", rp.Period().First(), rp.Period().Last())
		}
		time.Sleep(time.Minute)
	}
}

// stopClosedTimers ends, every 10 seconds, the timers whose site was closed
// and stayed closed, recording their time up to the close.
func stopClosedTimers(timers *application.TimerService) {
	for {
		if n, err := timers.StopClosed(context.Background(), time.Now()); err != nil {
			log.Printf("closed timers: %v", err)
		} else if n > 0 {
			log.Printf("ended %d timer(s) whose site was closed", n)
		}
		time.Sleep(10 * time.Second)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
