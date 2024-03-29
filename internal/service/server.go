package service

import (
	"database/sql"
	"github.com/ENSLERMAN/warehouse-back/internal/handlers"
	"github.com/ENSLERMAN/warehouse-back/internal/worker"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"time"
)

func StartServer() *gin.Engine {
	db := initDB()
	clickDB := initClickhouse()
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://enslerman.ru"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "x-requested-with", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return origin == "http://enslerman.ru"
		},
		MaxAge: 12 * time.Hour,
	}))

	accs := initBasicAuthLogins(db)
	go worker.UploadingHistory(clickDB, db)

	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "ping ok!")
	})
	nonAuth := r.Group("/auth")
	{
		nonAuth.POST("/register", handlers.Register(db))
		nonAuth.POST("/login", handlers.Login(db))
	}
	users := r.Group("/api/user", basicAuth(accs))
	{
		users.GET("/users:id", handlers.GetUserByID)
		users.POST("/update_role", handlers.UpdateRole(db))
		users.GET("/users", handlers.GetUsersByAccess(db))
		users.GET("/all", handlers.GetUsers(db))
		users.GET("/roles", handlers.GetRoles(db))
	}
	shipments := r.Group("/api/shipments", basicAuth(accs))
	{
		shipments.POST("/new_shipment", handlers.AddNewShipment(db))
		shipments.GET("/all", handlers.GetAllShipments(db))
		shipments.GET("/history", handlers.GetShipmentsHistory(db, clickDB))
	}
	dispatch := r.Group("/api/dispatch", basicAuth(accs))
	{
		dispatch.POST("/new_dispatch", handlers.AddNewDispatch(db))
		dispatch.POST("/close_dispatch", handlers.CloseDispatch(db))
		dispatch.GET("/all", handlers.GetDispatches(db))
		dispatch.GET("/products", handlers.GetProductsInDispatch(db))
		dispatch.GET("/history", handlers.GetHistoryDispatches(db, clickDB))
		dispatch.POST("/refuse", handlers.RefuseDispatch(db))
	}
	products := r.Group("/api/products", basicAuth(accs))
	{
		products.GET("/get", handlers.GetProducts(db))
		products.GET("/getByID", handlers.GetProductsByID(db))
		products.GET("/get_history", handlers.GetProductHistoryByID(db, clickDB))
		products.POST("/update", handlers.UpdateProduct(db))
		products.GET("/delete", handlers.DeleteProductsByID(db))
	}
	return r
}

func initBasicAuthLogins(db *sql.DB) gin.Accounts {
	accs := make(map[string]string)
	result, err := db.Query("select login, password from warehouse.users")
	if err != nil {
		logrus.Fatalf("cannot get users with err: %v", err.Error())
		return nil
	}
	var accounts struct {
		login    string
		password string
	}
	for result.Next() {
		err := result.Scan(&accounts.login, &accounts.password)
		if err != nil {
			logrus.Fatal("cannot get user " + err.Error())
			return nil
		}
		accs[accounts.login] = accounts.password
	}

	if err = result.Err(); err != nil {
		logrus.Error(err.Error())
		return nil
	}

	return accs
}
