package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"react-golang/src/backend/constants"
	"react-golang/src/backend/model"
	"react-golang/src/backend/utils"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sarulabs/di"
	"gorm.io/gorm"
)

type FunctionAPI interface {
	CreateFunction(c echo.Context) error
	FetchFunctionList(c echo.Context) error
	FetchFunctionDetail(c echo.Context) error
	DeleteFunction(c echo.Context) error
	RunFunction(c echo.Context) error
}

type FunctionAPIImpl struct {
	db *gorm.DB
}

func NewFunctionAPI(ioc di.Container) FunctionAPI {
	return FunctionAPIImpl{
		db: ioc.Get(constants.CONTAINER_DB_NAME).(*gorm.DB),
	}
}

type Caller struct {
	Data map[string]interface{}
}

type Function struct {
	Name     string                 `json:"name"`
	Action   string                 `json:"action"`
	Table    string                 `json:"table"`
	Multiple bool                   `json:"multiple"`
	Values   map[string]interface{} `json:"values"`
	Filter   []Filter               `json:"filter"`
	Columns  []string               `json:"columns"`
}

type functionReq struct {
	Name      string     `json:"name"`
	Functions []Function `json:"functions"`
}

func (f FunctionAPIImpl) CreateFunction(c echo.Context) error {
	var body *functionReq = new(functionReq)
	if err := c.Bind(body); err != nil {
		return c.JSON(http.StatusBadRequest, errors.New("Failed to bind: "+err.Error()))
	}

	// convert functions to json
	jsonFunc, err := json.Marshal(body.Functions)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	newFunction := model.FunctionStored{
		Name:     body.Name,
		Function: string(jsonFunc),
	}

	err = f.db.Model(&model.FunctionStored{}).Create(&newFunction).Error
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "success",
	})
}

func (f FunctionAPIImpl) FetchFunctionList(c echo.Context) error {
	var search *Search = new(Search)

	var functions []model.FunctionStored
	table := f.db.Select("name")
	if search.Search != "" {
		table = table.Where("name LIKE ?", fmt.Sprintf("%%%s%%", search.Search))
	}
	err := table.Find(&functions).Error
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, functions)
}

func (f FunctionAPIImpl) FetchFunctionDetail(c echo.Context) error {
	funcName := c.Param("func_name")

	var funcStored model.FunctionStored
	err := f.db.Where("name = ?", funcName).First(&funcStored).Error
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	var function functionReq
	function.Name = funcName
	err = json.Unmarshal([]byte(funcStored.Function), &function.Functions)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, function)
}

func (f FunctionAPIImpl) DeleteFunction(c echo.Context) error {
	funcName := c.Param("func_name")
	err := f.db.Model(&model.FunctionStored{}).Where("name = ?", funcName).Delete(&model.FunctionStored{}).Error
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, nil)
}

func (f FunctionAPIImpl) RunFunction(c echo.Context) error {
	funcName := c.Param("func_name")
	var function *model.FunctionStored

	paramUser := c.Get("user_id")
	var userID string
	if paramUser != nil {
		userID = paramUser.(string)
	}

	err := f.db.Model(&model.FunctionStored{}).Where("name = ?", funcName).First(&function).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "function does not exist",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	functions := []Function{}
	err = json.Unmarshal([]byte(function.Function), &functions)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	var caller *Caller = new(Caller)
	if err := c.Bind(caller); err != nil {
		return c.JSON(http.StatusBadRequest, errors.New("Failed to bind: "+err.Error()))
	}

	savedData := map[string]interface{}{}
	err = f.db.Transaction(func(db *gorm.DB) error {
		for _, f := range functions {
			switch f.Action {
			case "insert":
				if f.Multiple {
					bindedInput := BindMultipleInput(f.Values, caller.Data[f.Name].([]interface{}), savedData, userID)
					for i := range bindedInput {
						bindedInput[i]["id"], _ = utils.GenerateRandomString(16)
					}
					err := db.Table(f.Table).Create(bindedInput).Error
					if err != nil {
						return err
					}
				} else {
					bindedInput := BindSingularInput(f.Values, caller.Data[f.Name].(map[string]interface{}), savedData, userID)
					bindedInput["id"], _ = utils.GenerateRandomString(16)
					err := db.Table(f.Table).Create(bindedInput).Error
					if err != nil {
						return err
					}

					savedData[f.Name] = bindedInput["id"]
				}
			case "update":
				if f.Multiple {
					for _, input := range caller.Data[f.Name].([]map[string]interface{}) {
						filter := map[string]interface{}{
							"id = ?": input["id"],
						}

						bindedInput := BindSingularInput(f.Values, input, savedData, userID)
						table := db.Table(f.Table)
						for k, v := range filter {
							table = table.Where(k, v)
						}
						err := table.Updates(bindedInput).Error
						if err != nil {
							return err
						}
					}
				} else {
					data := caller.Data[f.Name].(map[string]interface{})
					filter := map[string]interface{}{
						"id = ?": data["id"],
					}

					bindedInput := BindSingularInput(f.Values, caller.Data[f.Name].(map[string]interface{}), savedData, userID)
					table := db.Table(f.Table)
					for k, v := range filter {
						table = table.Where(k, v)
					}
					err := table.Updates(bindedInput).Error
					if err != nil {
						return err
					}
				}
			case "delete":
				data := caller.Data[f.Name].(map[string]interface{})
				filter := map[string]interface{}{}

				for _, f := range f.Filter {
					if f.Value == "" {
						filter[f.Column+f.Operator] = data[f.Column]
					} else {
						filter[f.Column+f.Operator] = f.Value
					}
				}

				table := db.Table(f.Table)
				for k, v := range filter {
					table = table.Where(k, v)
				}
				err := table.Delete(nil).Error
				if err != nil {
					return err
				}
			case "fetch":
				result := []map[string]interface{}{}
				err := db.Table(f.Table).Select(f.Columns).Find(&result).Error
				if err != nil {
					return err
				}

				savedData[f.Name] = result
			}
		}

		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, savedData)
}

func applyFilter(query *gorm.DB, filter map[string]interface{}) *gorm.DB {
	for key, value := range filter {
		switch strings.ToLower(key) {
		case "and":
			for _, condition := range value.([]interface{}) {
				query = applyFilter(query, condition.(map[string]interface{}))
			}
		case "or":
			query = query.Where(func(tx *gorm.DB) *gorm.DB {
				ors := value.([]interface{})

				tx = applyFilter(tx, ors[0].(map[string]interface{}))
				for _, condition := range ors[1:] {
					for k, v := range condition.(map[string]interface{}) {
						tx = Or(query, k, v)
					}
				}

				return tx
			}(query))

			return query
		default:
			return Where(query, key, value)
		}
	}
	return query
}

func Where(query *gorm.DB, key string, value interface{}) *gorm.DB {
	return query.Where(fmt.Sprintf("%s = ?", key), value)
}

func Or(query *gorm.DB, key string, value interface{}) *gorm.DB {
	return query.Or(fmt.Sprintf("%s = ?", key), value)
}

func BindSingularInput(template map[string]interface{}, input map[string]interface{}, savedData map[string]interface{}, userID string) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range template {
		if strings.HasPrefix(v.(string), "$") {
			key := v.(string)[1:]
			if v == "$user.id" {
				result[k] = userID
			} else {
				result[k] = savedData[key]
			}
		} else {
			result[k] = input[k]
		}

	}

	return result
}

func BindMultipleInput(template map[string]interface{}, inputs []interface{}, savedData map[string]interface{}, userID string) []map[string]interface{} {
	result := []map[string]interface{}{}

	for _, input := range inputs {
		// currently testing, if broken, just change to the bottom one
		result = append(result, BindSingularInput(template, input.(map[string]interface{}), savedData, userID))

		/* ================================================================== */
		// current := map[string]interface{}{}
		// for k, v := range template {
		// 	if strings.HasPrefix(v.(string), "$") {
		// 		key := v.(string)[1:]
		// 		if v == "$user.id" {
		// 			current[k] = "[this_is_user_id]"
		// 		} else {
		// 			current[k] = savedData[key]
		// 		}
		// 	} else {
		// 		current[k] = input[k]
		// 	}
		// }
		// result = append(result, current)
	}

	return result
}
