package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type Admin struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Salt      string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (a *Admin) TableName() string {
	return "_admin"
}

type Index struct {
	Name    string   `json:"name"`
	Indexes []string `json:"indexes"`
}

type Tables struct {
	Name        string  `json:"name,omitempty" gorm:"primaryKey"`
	Auth        bool    `json:"auth,omitempty" gorm:"column:auth"`
	System      bool    `json:"system,omitempty" gorm:"column:system"`
	Indexes     string  `json:"indexes,omitempty" gorm:"column:indexes"`
	SystemIndex []Index `json:"index,omitempty" gorm:"-"`
	ViewRule    string  `json:"view_rule,omitempty" gorm:"column:view_rule;default:ADMIN_ONLY"`
	ReadRule    string  `json:"read_rule,omitempty" gorm:"column:read_rule;default:ADMIN_ONLY"`
	InsertRule  string  `json:"insert_rule,omitempty" gorm:"column:insert_rule;default:ADMIN_ONLY"`
	UpdateRule  string  `json:"update_rule,omitempty" gorm:"column:update_rule;default:ADMIN_ONLY"`
	DeleteRule  string  `json:"delete_rule,omitempty" gorm:"column:delete_rule;default:ADMIN_ONLY"`
}

func (t *Tables) TableName() string {
	return "_table"
}

type QueryHistory struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Query     string    `json:"query"`
	CreatedAt time.Time `json:"created_at"`
}

func (q *QueryHistory) TableName() string {
	return "_queryHistory"
}

type FunctionStored struct {
	Name     string `json:"name" gorm:"primaryKey"`
	Function string `json:"function" gorm:"column:function"`
}

func (f *FunctionStored) TableName() string {
	return "_function"
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&Admin{}, &Tables{}, &QueryHistory{}, &FunctionStored{})
	if err != nil {
		return err
	}

	databases := []Tables{
		{Name: "_admin", Auth: true, System: true},
		{Name: "_queryHistory", Auth: false, System: true},
		{Name: "_function", Auth: false, System: true},
	}
	err = db.Model(&Tables{}).Create(databases).Error
	if err != nil {
		return err
	}

	return err
}

// OTHERS MODELS

type Column struct {
	CID       int    `json:"cid"`
	Default   string `json:"dflt_value"`
	Name      string `json:"name"`
	NotNull   int    `json:"notnull"`
	PK        int    `json:"pk"`
	Type      string `json:"type"`
	Reference string `json:"reference,omitempty"`
}

type Field struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Nullable  bool   `json:"nullable"`
	Reference string `json:"reference,omitempty"`
	Unique    bool   `json:"unique"`
}

func (f *Field) ConvertTypeToSQLiteType() string {
	switch strings.ToLower(f.Type) {
	case "text", "string":
		return "TEXT"
	case "number", "real":
		return "REAL"
	case "boolean":
		return "BOOLEAN"
	case "datetime", "timestamp":
		return "DATETIME"
	case "file", "blob":
		return "BLOB"
	case "relation":
		return "RELATION"
	default:
		return ""
	}
}

type CreateTable struct {
	Name    string  `json:"table_name"`
	Fields  []Field `json:"fields"`
	Indexes []Index `json:"indexes"`
	Type    string  `json:"table_type"`
}
