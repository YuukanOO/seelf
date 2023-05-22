package main

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("DSN")

	if dsn == "" {
		panic("db connection string should be set with the DSN env variable!")
	}

	db, err := sql.Open("postgres", dsn)

	if err != nil {
		panic(err)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS runs (
	id SERIAL PRIMARY KEY,
	date TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

INSERT INTO runs(date) VALUES (now());
	`); err != nil {
		panic(err)
	}

	r := gin.Default()

	r.GET("/", func(ctx *gin.Context) {
		rows, err := db.Query("SELECT id, date FROM runs")

		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		defer rows.Close()

		var results []data

		for rows.Next() {
			var d data

			if err := rows.Scan(&d.ID, &d.Date); err != nil {
				ctx.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			results = append(results, d)
		}

		ctx.JSON(http.StatusOK, response{
			Description: "Simple application which logs in a database the date of each run and exposes environment variables to test seelf.",
			Env:         os.Environ(),
			Runs:        results,
		})
	})

	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}

type data struct {
	ID   int       `json:"id"`
	Date time.Time `json:"date"`
}

type response struct {
	Description string   `json:"description"`
	Env         []string `json:"env"`
	Runs        []data   `json:"runs"`
}
