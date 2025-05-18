package main

import (
	"cmp"
	"context"
	"encoding/xml"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/0queue/todoist-rss/internal/rss"
	"github.com/0queue/todoist-rss/internal/todoist"
)

var linkRegex = regexp.MustCompile(`\[(.*)\]\((.*)\)`)

func main() {
	signalCtx, signalCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer signalCancel()

	client := todoist.New(os.Getenv("TOKEN"))

	mux := http.NewServeMux()
	mux.Handle("GET /rss.xml", requestLogger(xmlHandler(client, cmp.Or(os.Getenv("LABEL"), "rss"))))

	srv := http.Server{
		Addr:    cmp.Or(os.Getenv("ADDR"), ":8080"),
		Handler: mux,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Error starting http server", slog.Any("error", err))
			signalCancel()
		}
	}()

	slog.Info("Server running", slog.String("addr", srv.Addr))

	<-signalCtx.Done()

	slog.Info("Shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	err := srv.Shutdown(shutdownCtx)
	if err != nil {
		slog.Error("Error shutting down http server", slog.Any("error", err))
	}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()

		next.ServeHTTP(w, r)

		slog.Info(
			"Request",
			slog.String("path", r.URL.Path),
			slog.String("d", time.Now().Sub(t).String()),
		)
	})
}

func xmlHandler(client *todoist.Client, label string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ts, err := client.GetTasks(r.Context(), label)
		if err != nil {
			slog.Error("Error getting tasks", slog.Any("error", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		items := make([]rss.Item, 0)
		for _, t := range ts {
			matches := linkRegex.FindStringSubmatch(t.Content)
			if len(matches) >= 3 {
				items = append(items, rss.Item{
					Title: matches[1],
					Link:  matches[2],
					Description: &rss.Description{
						Text: t.Description,
					},
					Guid:    "todoist-" + t.ID,
					PubDate: t.AddedAt.Format(time.RFC822),
				})
			} else {
				items = append(items, rss.Item{
					Title: t.Content,
					Link:  "https://app.todoist.com/app/task/" + t.ID,
					Description: &rss.Description{
						Text: t.Description,
					},
					Guid:    "todoist-" + t.ID,
					PubDate: t.AddedAt.Format(time.RFC822),
				})
			}
		}

		rr := rss.Rss{
			Version: "2.0",
			Channel: rss.Channel{
				Title:       "Todoist RSS",
				Link:        "https://app.todoist.com",
				Description: "RSS items generated from Todoist tasks",
				Items:       items,
			},
		}

		bs, err := xml.MarshalIndent(rr, "", "  ")
		if err != nil {
			slog.Error("Error marshalling response", slog.Any("error", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write(bs)
		if err != nil {
			slog.Error("Failed to write response", slog.Any("error", err))
			return
		}

		// close the tasks if miniflux makes the request
		if strings.Contains(r.UserAgent(), "+https://miniflux.app") {
			errs := make([]error, 0)
			for _, t := range ts {
				err := client.CloseTask(r.Context(), t.ID)
				if err != nil {
					errs = append(errs, err)
				}

				slog.Info("closed task", slog.String("taskId", t.ID))
			}
			err := errors.Join(errs...)
			if err != nil {
				slog.Error("Failed to close all tasks", slog.Any("error", err))
				return
			}
		}
	}
}
