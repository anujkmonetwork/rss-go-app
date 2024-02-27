package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/anuj070894/rssagg/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	DB *database.Queries
}

func main() {
	fmt.Println("Hello world!")
	feed, err := urlToFeed("https://wagslane.dev/index.xml")

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(feed)
	godotenv.Load()

	dbUrl := os.Getenv("DB_URL")

	if dbUrl == "" {
		log.Fatal("DB_URL is not found in the environment")
	}

	conn, err := sql.Open("postgres", dbUrl)

	if err != nil {
		log.Fatal("Can't connect to database: ", err)
	}

	queries := database.New(conn)

	apiConfig := apiConfig{
		DB: queries,
	}

	go startScraping(queries, 10, time.Minute)
	port := os.Getenv("PORT")

	router := chi.NewRouter()

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	v1Router := chi.NewRouter()

	v1Router.Get("/healthz", handlerReadiness)

	v1Router.Get("/err", handlerErr)

	v1Router.Post("/users", apiConfig.handleCreateUser)
	v1Router.Get("/users", apiConfig.middlewareAuth(apiConfig.handleGetUser))

	v1Router.Post("/feeds", apiConfig.middlewareAuth(apiConfig.handleCreateFeed))
	v1Router.Get("/feeds", apiConfig.handleGetFeeds)

	v1Router.Post("/feed_follows", apiConfig.middlewareAuth(apiConfig.handleCreateFeedFollow))
	v1Router.Get("/feed_follows", apiConfig.middlewareAuth(apiConfig.handleGetFeedFollows))
	v1Router.Delete("/feed_follows/{feedFollowID}", apiConfig.middlewareAuth(apiConfig.handleDeleteFeedFollows))

	v1Router.Get("/posts", apiConfig.middlewareAuth(apiConfig.handleGetPostsForUser))

	router.Mount("/v1", v1Router)

	srv := &http.Server{
		Handler: router,
		Addr:    ":" + port,
	}

	log.Printf("Server starting on port %v", port)

	err = srv.ListenAndServe()

	if err != nil {
		log.Fatal(err)
	}
}
