package routes

import (
	"github.com/gin-gonic/gin"

	"stratus/config"
	"stratus/handlers"
	"stratus/middleware"
	"stratus/services"
)

func SetupRoutes(r *gin.Engine, cfg *config.Config) {
	storageService := services.NewStorageService(cfg)
	storageService.InitStorage()

	authHandler := handlers.NewAuthHandler(cfg)
	fileHandler := handlers.NewFileHandler(cfg, storageService)
	adminHandler := handlers.NewAdminHandler(cfg)
	webdavHandler := handlers.NewWebDAVHandler(cfg, storageService)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "stratus"})
	})

	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware(cfg))
	{
		api.POST("/auth/logout", authHandler.Logout)
		api.GET("/auth/me", authHandler.Me)
		api.PUT("/auth/profile", authHandler.UpdateProfile)
		api.PUT("/auth/password", authHandler.ChangePassword)

		files := api.Group("/files")
		{
			files.GET("", fileHandler.List)
			files.GET("/:id", fileHandler.Get)
			files.GET("/:id/contents", fileHandler.GetContents)
			files.POST("/upload", fileHandler.Upload)
			files.GET("/:id/download", fileHandler.Download)
			files.POST("/folder", fileHandler.CreateFolder)
			files.PUT("/:id/rename", fileHandler.Rename)
			files.PUT("/:id/move", fileHandler.Move)
			files.POST("/:id/copy", fileHandler.Copy)
			files.DELETE("/:id/trash", fileHandler.Trash)
			files.POST("/:id/restore", fileHandler.Restore)
			files.DELETE("/:id", fileHandler.Delete)
			files.GET("/search", fileHandler.Search)
		}

		trash := api.Group("/trash")
		{
			trash.GET("", fileHandler.ListTrash)
			trash.DELETE("", fileHandler.EmptyTrash)
		}

		api.GET("/storage/stats", fileHandler.StorageStats)

		admin := api.Group("/admin")
		admin.Use(middleware.AdminMiddleware())
		{
			admin.GET("/users", adminHandler.ListUsers)
			admin.GET("/users/:id", adminHandler.GetUser)
			admin.PUT("/users/:id", adminHandler.UpdateUser)
			admin.DELETE("/users/:id", adminHandler.DeleteUser)
			admin.GET("/stats", adminHandler.SystemStats)
			admin.GET("/activities", adminHandler.ListActivities)
		}
	}

	webdav := r.Group("/webdav")
	webdav.Use(middleware.BasicAuthMiddleware(cfg))
	{
		webdav.Handle("OPTIONS", "", webdavHandler.Options)
		webdav.Handle("PROPFIND", "", webdavHandler.Propfind)
		webdav.Handle("OPTIONS", "/*path", webdavHandler.Options)
		webdav.Handle("PROPFIND", "/*path", webdavHandler.Propfind)
		webdav.GET("/*path", webdavHandler.Get)
		webdav.PUT("/*path", webdavHandler.Put)
		webdav.Handle("MKCOL", "/*path", webdavHandler.Mkcol)
		webdav.DELETE("/*path", webdavHandler.Delete)
		webdav.Handle("MOVE", "/*path", webdavHandler.Move)
		webdav.Handle("COPY", "/*path", webdavHandler.Copy)
		webdav.HEAD("/*path", webdavHandler.Head)
		webdav.Handle("LOCK", "/*path", webdavHandler.Lock)
		webdav.Handle("UNLOCK", "/*path", webdavHandler.Unlock)
	}
}
