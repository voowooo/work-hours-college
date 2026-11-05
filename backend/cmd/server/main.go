package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/rs/cors"
	"github.com/workhourscounter/backend/graph"
	"github.com/workhourscounter/backend/internal/db"
	"github.com/workhourscounter/backend/internal/schedule"
	"github.com/workhourscounter/backend/pkg/auth"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           86400,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/schedule", schedule.Handler)

	database, err := db.New()
	if err != nil {
		log.Printf("Database unavailable, schedule endpoint only: %v", err)
	} else {
		defer database.Close()

		resolver := &graph.Resolver{DB: database}
		srv := handler.NewDefaultServer(
			graph.NewExecutableSchema(graph.Config{
				Resolvers: resolver,
			}),
		)
		mux.Handle("/query", auth.Middleware(srv))
		mux.Handle("/", playground.Handler("GraphQL Playground", "/query"))
	}

	log.Printf("Server running on http://localhost:%s/", port)
	log.Printf("Playground: http://localhost:%s/", port)
	log.Fatal(http.ListenAndServe(":"+port, c.Handler(mux)))
}
