package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"react-golang/src/backend/constants"
	"react-golang/src/backend/model"
	"react-golang/src/backend/service"
	"react-golang/src/backend/utils"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sarulabs/di"
	"gorm.io/gorm"
)

type DatabaseAPI interface {
	FetchAllTables(c echo.Context) error
	FetchTableColumns(c echo.Context) error
	FetchRows(c echo.Context) error

	CreateTable(c echo.Context) error
	FetchDataByID(c echo.Context) error
	InsertData(c echo.Context) error
	UpdateData(c echo.Context) error
	DeleteData(c echo.Context) error
	DeleteTable(c echo.Context) error

	RunQuery(c echo.Context) error
	FetchQueryHistory(c echo.Context) error

	Backup(c echo.Context) error
	Restore(c echo.Context) error
	FetchBackups(c echo.Context) error
}

type DatabaseAPIImpl struct {
	db      *gorm.DB
	service *service.Service
}

func NewDatabaseAPI(ioc di.Container) DatabaseAPI {
	return &DatabaseAPIImpl{
		db:      ioc.Get(constants.CONTAINER_DB_NAME).(*gorm.DB),
		service: ioc.Get(constants.CONTAINER_SERVICE).(*service.Service),
	}
}

type DBResult []map[string]interface{}

func (d *DatabaseAPIImpl) FetchAllTables(c echo.Context) error {
	var result []map[string]interface{} = make([]map[string]interface{}, 0)

	var params *Search = new(Search)
	if err := (&echo.DefaultBinder{}).BindQueryParams(c, params); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	query := d.db.Model(&model.Tables{}).
		Select("name, is_auth").
		Where("is_system = ?", false).
		Order("name ASC")

	if params.Search != "" {
		query = query.Where("name LIKE ?", fmt.Sprintf("%%%s%%", params.Search))
	}

	err := query.Find(&result).Error
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

type fetchColumn struct {
	FetchAuthColumn bool `json:"fetch_auth_column" query:"fetch_auth_column"`
}

func (d *DatabaseAPIImpl) FetchTableColumns(c echo.Context) error {
	tableName := c.Param("table_name")

	var params *fetchColumn = new(fetchColumn)
	if err := (&echo.DefaultBinder{}).BindQueryParams(c, params); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	result, err := d.service.Table.Columns(tableName, params.FetchAuthColumn)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

type fetchRowsParam struct {
	Filter   string `query:"filter"`
	Sort     string `query:"sort"`
	Page     int    `query:"page"`
	PageSize int    `query:"page_size"`
}

type fetchRowsRes struct {
	Data      []map[string]interface{} `json:"data"`
	Page      int                      `json:"page"`
	PageSize  int                      `json:"page_size"`
	TotalData int64                    `json:"total_data"`
}

func (d *DatabaseAPIImpl) FetchRows(c echo.Context) error {
	tableName := c.Param("table_name")
	var res fetchRowsRes

	table, err := d.service.Table.Info(tableName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	var params *fetchRowsParam = new(fetchRowsParam)
	if err := (&echo.DefaultBinder{}).BindQueryParams(c, params); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	columns := "*"
	if table.IsAuth {
		allColumn := []model.Column{}
		err = d.db.Raw(fmt.Sprintf("PRAGMA table_info(%s)", tableName)).
			Scan(&allColumn).
			Error

		if err != nil {
			return err
		}

		columns = ""

		for _, column := range allColumn {
			if column.Name != "password" && column.Name != "salt" {
				if columns != "" {
					columns = fmt.Sprintf("%s, %s", columns, column.Name)
				} else {
					columns = column.Name
				}
			}
		}
	}

	paramUser := c.Get("user_id")
	var userID string
	if paramUser != nil {
		userID = paramUser.(string)
	}

	rawQuery := `
	SELECT %s FROM %s
	`
	query := fmt.Sprintf(rawQuery, columns, tableName)

	if params.Filter != "" {
		if strings.Contains(params.Filter, "$user.id") {
			params.Filter = strings.ReplaceAll(params.Filter, "$user.id", userID)
		}
		query = query + `WHERE ` + params.Filter
	}
	if params.Sort != "" {
		query = query + ` ORDER BY ` + params.Sort
	}
	if params.Page != 0 && params.PageSize != 0 {
		query = query + ` LIMIT ` + strconv.Itoa(params.PageSize) + ` OFFSET ` + strconv.Itoa((params.Page-1)*params.PageSize)
	}

	res.Data = make([]map[string]interface{}, 0)
	if err := d.db.Raw(query).
		Find(&res.Data).
		Error; err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	rawCountQuery := `
	SELECT COUNT(*) FROM %s
	`
	query = fmt.Sprintf(rawCountQuery, tableName)
	if params.Filter != "" {
		query = query + `WHERE ` + params.Filter
	}
	if err := d.db.Raw(query).First(&res.TotalData).Error; err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	res.Page = params.Page
	res.PageSize = params.PageSize

	return c.JSON(http.StatusOK, res)
}

type fields struct {
	FieldType    string `json:"field_type"`
	FieldName    string `json:"field_name"`
	Nullable     bool   `json:"nullable"`
	RelatedTable string `json:"related_table,omitempty"`
	Indexed      bool   `json:"indexed"`
	Unique       bool   `json:"unique"`
}

func (f *fields) convertTypeToSQLiteType() string {
	switch f.FieldType {
	case "text":
		return "TEXT"
	case "number":
		return "REAL"
	case "boolean":
		return "BOOLEAN"
	case "datetime":
		return "DATETIME"
	case "file":
		return "BLOB"
	case "relation":
		return "RELATION"
	default:
		return ""
	}
}

type createTableReq struct {
	TableName string   `json:"table_name"`
	IDType    string   `json:"id_type"`
	Fields    []fields `json:"fields"`
	Type      string   `json:"table_type"`
}

func (d *DatabaseAPIImpl) CreateTable(c echo.Context) error {
	var params *createTableReq = new(createTableReq)
	if err := c.Bind(&params); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	id := "id %s"

	switch params.IDType {
	case "string":
		id = fmt.Sprintf(id, "TEXT PRIMARY KEY DEFAULT (hex(randomblob(8)))")
	case "manual":
		id = fmt.Sprintf(id, "TEXT PRIMARY KEY")
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid id type")
	}

	fields := []string{
		id,
	}

	isAuth := false

	if params.Type == "users" {
		authFields := []string{
			"email TEXT NOT NULL",
			"password TEXT NOT NULL",
			"salt TEXT NOT NULL",
		}
		isAuth = true

		fields = append(fields, authFields...)
	}

	foreignKeys := []string{}
	uniques := []string{}
	indexes := []string{}

	for i := 0; i < len(params.Fields); i++ {
		dtype := params.Fields[i].convertTypeToSQLiteType()
		// IGNORE UNSUPPORTED DATATYPES FOR NOW
		if dtype == "" {
			continue
		}

		var field string
		if dtype == "RELATION" {
			field = fmt.Sprintf("%s %s", params.Fields[i].FieldName, "TEXT")
			foreignKeys = append(foreignKeys, fmt.Sprintf("FOREIGN KEY(%s) REFERENCES %s(id) ON UPDATE CASCADE", params.Fields[i].FieldName, params.Fields[i].RelatedTable))
		} else {
			field = fmt.Sprintf("%s %s", params.Fields[i].FieldName, dtype)
		}

		if !params.Fields[i].Nullable {
			field += " NOT NULL"
		}

		if params.Fields[i].Indexed {
			indexes = append(indexes, fmt.Sprintf("CREATE INDEX idx_%s ON %s (%s)", params.Fields[i].FieldName, params.TableName, params.Fields[i].FieldName))
		}

		if params.Fields[i].Unique {
			uniques = append(uniques, fmt.Sprintf("UNIQUE (%s)", params.Fields[i].FieldName))
		}

		fields = append(fields, field)
	}

	fields = append(fields, []string{
		"created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
		"updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
	}...)

	fields = append(append(fields, uniques...), foreignKeys...)

	query := `
		CREATE TABLE %s (
			%s
		)
	`

	query = fmt.Sprintf(query, params.TableName, strings.Join(fields, ","))

	err := d.db.Transaction(func(tx *gorm.DB) error {
		err := d.db.Exec(query).Error
		if err != nil {
			return err
		}

		// add index
		for _, index := range indexes {
			err = d.db.Exec(index).Error
			if err != nil {
				return err
			}
		}

		// check if trigger already exist
		var triggerHolder int64
		err = d.db.Table("sqlite_master").
			Select("*").
			Where("type = ?", "trigger").
			Where("name = ?", fmt.Sprintf("updated_timestamp_%s", params.TableName)).
			Count(&triggerHolder).Error
		if err != nil {
			return err
		}

		// add trigger to update updated_at value on update
		if triggerHolder == 0 {
			err = d.db.Exec(fmt.Sprintf(`
			CREATE TRIGGER updated_timestamp_%s
			AFTER UPDATE ON %s
			FOR EACH ROW
			BEGIN
				UPDATE %s SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
			END
			`, params.TableName, params.TableName, params.TableName)).Error
			if err != nil {
				return err
			}
		}
		err = d.db.Create(
			&model.Tables{
				Name:     params.TableName,
				IsAuth:   isAuth,
				IsSystem: false,
			}).
			Error
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, nil)
}

func (d *DatabaseAPIImpl) FetchDataByID(c echo.Context) error {
	tableName := c.Param("table_name")
	id := c.Param("id")
	var result map[string]interface{} = make(map[string]interface{}, 0)

	if err := d.db.Table(tableName).
		Select("*").
		Where("id = ?", id).
		Find(&result).
		Limit(1).
		Error; err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

func (d *DatabaseAPIImpl) InsertData(c echo.Context) error {
	tableName := c.Param("table_name")

	err := c.Request().ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to parse multipart form",
		})
	}

	table, err := d.service.Table.Info(tableName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}
	if table.IsAuth {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Insertion to user type table can only be done through auth API",
		})
	}

	filteredData := make(map[string]interface{})

	id, _ := utils.GenerateRandomString(16)
	form := c.Request().MultipartForm

	for k, v := range form.Value {
		if len(v) == 0 || k == "id" || k == "created_at" || k == "updated_at" {
			continue
		}
		if v[0] == "" {
			continue
		}
		if v[0] == "$user.id" {
			if c.Get("user_id") == nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "User not authorized",
				})
			}
			filteredData[k] = c.Get("user_id")
		}
		filteredData[k] = v[0]
	}

	for k, files := range form.File {
		file, err := files[0].Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to open file",
			})
		}

		defer file.Close()

		newFileName := utils.GenerateUUIDV7()
		fileExtension := filepath.Ext(files[0].Filename)

		storageDir := filepath.Join("..", "public", newFileName+fileExtension)
		err = d.service.Storage.Save(file, storageDir)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": err.Error(),
			})
		}

		filteredData[k] = fmt.Sprintf("%s%s", newFileName, fileExtension)
		continue
	}

	filteredData["id"] = id

	d.service.Table.Insert(tableName, filteredData)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "success",
	})
}

func (d *DatabaseAPIImpl) UpdateData(c echo.Context) error {
	tableName := c.Param("table_name")

	err := c.Request().ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to parse multipart form",
		})
	}

	updatedData := make(map[string]interface{})
	form := c.Request().MultipartForm

	for k, v := range form.Value {
		if len(v) == 0 || k == "created_at" || k == "updated_at" {
			continue
		}
		if v[0] == "" {
			continue
		}
		updatedData[k] = v[0]
	}

	for k, files := range form.File {
		file, err := files[0].Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to open file",
			})
		}

		defer file.Close()

		newFileName := utils.GenerateUUIDV7()
		fileExtension := filepath.Ext(files[0].Filename)
		storageDir := filepath.Join("..", "public", newFileName+fileExtension)
		_, err = os.Stat(storageDir)
		if os.IsExist(err) {
			os.Remove(storageDir)
		}

		dst, err := os.Create(storageDir)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to create destination file",
			})
		}
		defer dst.Close()

		// Copy file to destination
		if _, err := io.Copy(dst, file); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to save file",
			})
		}

		updatedData[k] = fmt.Sprintf("%s%s", newFileName, fileExtension)
		continue
	}

	err = d.service.Table.Update(tableName, updatedData)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "success",
	})
}

type deleteDataReq struct {
	ID []string `json:"id"`
}

func (d *DatabaseAPIImpl) DeleteData(c echo.Context) error {
	tableName := c.Param("table_name")

	var params *deleteDataReq = new(deleteDataReq)
	if err := c.Bind(&params); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	err := d.service.Table.BatchDelete(tableName, params.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, nil)
}

type queryReq struct {
	Query string
}

func (d *DatabaseAPIImpl) RunQuery(c echo.Context) error {
	var params *queryReq = new(queryReq)
	if err := c.Bind(&params); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	var result []map[string]interface{} = make([]map[string]interface{}, 0)

	rows, err := d.db.Raw(params.Query).Rows()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer rows.Close()

	for rows.Next() {
		var row map[string]interface{}
		if err := d.db.ScanRows(rows, &row); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": err.Error(),
			})
		}
		result = append(result, row)
	}

	go func(query string) {
		d.db.Create(&model.QueryHistory{
			Query: query,
		})

		d.db.Exec(`
		DELETE FROM query_history
		WHERE id NOT IN (
			SELECT id
			FROM (
				SELECT id
				FROM query_history
				ORDER BY id DESC
				LIMIT 10
			)
		);
		`)
	}(params.Query)

	return c.JSON(http.StatusOK, result)
}

func (d *DatabaseAPIImpl) FetchQueryHistory(c echo.Context) error {
	var queryHistories []model.QueryHistory

	result := d.db.Limit(10).Order("id DESC").Find(&queryHistories)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": result.Error.Error(),
		})
	}

	return c.JSON(http.StatusOK, queryHistories)
}

func (d *DatabaseAPIImpl) DeleteTable(c echo.Context) error {
	tableName := c.Param("table_name")

	err := d.db.Transaction(func(tx *gorm.DB) error {
		err := d.db.Exec(fmt.Sprintf("DROP TABLE %s", tableName)).Error
		if err != nil {
			return err
		}

		err = d.db.
			Where("lower(name) = ?", strings.ToLower(tableName)).
			Delete(&model.Tables{}).
			Error
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, nil)
}

func (d *DatabaseAPIImpl) Backup(c echo.Context) error {
	err := d.service.Backup.Backup()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "success",
	})
}

func (d *DatabaseAPIImpl) Restore(c echo.Context) error {
	filename := c.Param("filename")

	err := d.service.Backup.Restore(filename)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "success",
	})
}

func (d *DatabaseAPIImpl) FetchBackups(c echo.Context) error {
	backupPath := os.Getenv("BACKUP_PATH")

	datas, err := os.ReadDir(backupPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	filenames := []string{}
	for _, data := range datas {
		filenames = append(filenames, data.Name())
	}

	return c.JSON(http.StatusOK, filenames)
}
