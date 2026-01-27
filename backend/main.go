package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"stratus/config"
	"stratus/database"
	"stratus/middleware"
	"stratus/models"
	"stratus/routes"
)

func main() {
	cfg := config.Load()

	if err := database.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	createInitialAdmin()

	r := gin.Default()

	r.Use(middleware.CORSMiddleware())
	r.Use(gin.Recovery())

	routes.SetupRoutes(r, cfg)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		database.Close()
		os.Exit(0)
	}()

	log.Printf("Stratus server starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func createInitialAdmin() {
	var count int64
	database.DB.Table("users").Count(&count)
	if count == 0 {
		log.Println("Creating initial admin user...")

		adminPassword := os.Getenv("ADMIN_PASSWORD")
		if adminPassword == "" {
			adminPassword = "admin123"
		}

		admin := models.User{
			Username:    "admin",
			Email:       "admin@stratus.local",
			DisplayName: "Administrator",
			IsAdmin:     true,
			IsActive:    true,
			Quota:       107374182400, // 100GB
		}
		admin.SetPassword(adminPassword)

		if err := database.DB.Create(&admin).Error; err != nil {
			log.Printf("Failed to create admin user: %v", err)
			return
		}

		rootFolder := models.File{
			Name:        "root",
			Path:        "/",
			IsDirectory: true,
			OwnerID:     admin.ID,
			StoragePath: admin.ID.String(),
		}
		database.DB.Create(&rootFolder)

		log.Printf("Initial admin user created: username=admin, password=%s", adminPassword)
		log.Println("Please change the admin password after first login!")
	}
}
