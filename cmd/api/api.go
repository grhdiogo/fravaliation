package main

import (
	"fmt"
	"fravaliation/internal/infra/db/postgres"
	"fravaliation/internal/interf"
	"fravaliation/internal/interf/wa"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
)

func main() {
	// port
	port := os.Getenv("PORT")
	if port == "" {
		port = "18080"
	}
	// log
	log.Printf("Api initialized on PORT: %s", port)
	pgconfig := postgres.Config{
		Host: os.Getenv("DATABASE_HOST"),
		User: os.Getenv("POSTGRES_USER"),
		Port: os.Getenv("DATABASE_PORT"),
		DBNm: os.Getenv("POSTGRES_DB"),
		Pswd: os.Getenv("POSTGRES_PASSWORD"),
	}
	fmt.Println(">>>>>>>>>", pgconfig)
	postgres.SetConfiguration(pgconfig)
	inst := postgres.GetInstance()
	log.Print("Initialize Migrations")
	err := inst.Init()
	if err != nil {
		panic(err)
	}
	waApp := wa.NewWebApplication(interf.WebServiceConfig{
		Prefix:  "a",
		Version: "v0",
	})
	//server setup
	srv := &http.Server{
		Handler: handlers.LoggingHandler(os.Stdout, waApp.GetRouter()),
		Addr:    fmt.Sprintf("0.0.0.0:%s", port),
		//Good practice: enforce timeouts for servers you create!
		WriteTimeout: 800 * time.Second,
		ReadTimeout:  800 * time.Second,
	}
	//log initializing webapp
	log.Printf("Initializing webapp")
	//log
	log.Fatal(srv.ListenAndServe())
}
