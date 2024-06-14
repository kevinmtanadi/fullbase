package api

import (
	api_function "react-golang/src/backend/api/functions"

	"github.com/labstack/echo/v4"
	"github.com/sarulabs/di"
)

type API struct {
	app      *echo.Echo
	router   *echo.Group
	Admin    AdminAPI
	Database DatabaseAPI
	Function api_function.FunctionAPI
}

type Search struct {
	Search string `json:"search" query:"search"`
}

func NewAPI(app *echo.Echo, ioc di.Container) *API {
	return &API{
		app:      app,
		router:   app.Group("/api"),
		Admin:    NewAdminAPI(ioc),
		Database: NewDatabaseAPI(ioc),
		Function: api_function.NewFunctionAPI(ioc),
	}
}

func (api *API) Serve() {
	api.DbAPI()
	api.FunctionAPI()
}

func (api *API) DbAPI() {
	dbRouter := api.router.Group("/db")

	dbRouter.GET("/tables", api.Database.FetchAllTables)
	dbRouter.POST("/query", api.Database.RunQuery)
	dbRouter.GET("/columns/:table_name", api.Database.FetchTableColumns)
	dbRouter.POST("/rows/:table_name", api.Database.FetchRows)
	dbRouter.GET("/table/:table_name/:id", api.Database.FetchDataByID)
	dbRouter.POST("/table/create", api.Database.CreateTable)
	dbRouter.POST("/row/insert", api.Database.InsertData)
	dbRouter.PUT("/row/update", api.Database.UpdateData)
	dbRouter.DELETE("/row/:table_name/:id", api.Database.DeleteData)
	dbRouter.DELETE("/table/:table_name", api.Database.DeleteTable)
}

func (api *API) AdminAPI() {
	adminRouter := api.router.Group("/admin")

	adminRouter.POST("/register", api.Admin.RegisterNewAdmin)
}

func (api *API) FunctionAPI() {
	functionRouter := api.router.Group("/function")

	functionRouter.POST("/run", api.Function.RunFunction)
}
