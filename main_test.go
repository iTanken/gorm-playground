package main

import (
	"reflect"
	"testing"

	"gorm.io/gorm"
)

// GORM_REPO: https://github.com/iTanken/gorm.git
// GORM_BRANCH: master
// TEST_DRIVERS: sqlite, mysql, postgres, sqlserver

func TestGORM(t *testing.T) {
	user := User{Name: `"jin'zhu"`} // string values contain single or double quotes

	// SQLite:
	// INSERT INTO `users` (`created_at`,`updated_at`,`deleted_at`,`name`,`age`,`birthday`,`company_id`,`manager_id`,`active`)
	// VALUES ("2023-12-27 17:58:17.329","2023-12-27 17:58:17.329",NULL,
	//   """jin'zhu""",
	//    0,NULL,NULL,NULL,false
	// ) RETURNING `id`
	DB.Create(&user)

	var result User
	if err := DB.First(&result, user.ID).Error; err != nil {
		t.Errorf("Failed, got error: %v", err)
	}
}

func TestExplain(t *testing.T) {
	type args struct {
		prepareSql string
		values     []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantSQL string
	}{
		{"mysql", args{"SELECT ? AS QUOTES_STR", []interface{}{"'"}}, `SELECT '''' AS QUOTES_STR`},
		{"postgres", args{"SELECT $1 AS QUOTES_STR", []interface{}{"'"}}, `SELECT '''' AS QUOTES_STR`},
		{"sqlserver", args{"SELECT @p1 AS QUOTES_STR", []interface{}{"'"}}, `SELECT '''' AS QUOTES_STR`},
		{"sqlite", args{"SELECT ? AS QUOTES_STR", []interface{}{`"`}}, `SELECT """" AS QUOTES_STR`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if name := DB.Dialector.Name(); name != tt.name {
				t.Logf("%s skip %s...", name, tt.name)
				return
			}
			gotSQL := DB.Dialector.Explain(tt.args.prepareSql, tt.args.values...)
			if reflect.DeepEqual(gotSQL, tt.wantSQL) {
				var result string
				if err := DB.Raw(gotSQL).Row().Scan(&result); err == nil {
					t.Logf("exec `%s` result = `%s`", gotSQL, result)
				} else {
					t.Errorf("exec `%s` got error: %v", gotSQL, err)
				}
			} else {
				t.Errorf("Explain gotSQL = %v, want %v", gotSQL, tt.wantSQL)
			}
		})
	}
}

type UserPreloadWithSpecificModel struct {
	gorm.Model
	Username string
	Orders   []OrderPreloadWithSpecificModel `gorm:"foreignKey:UserID;references:ID"`
}

type CustomUserRes struct {
	ID       uint                            `gorm:"primaryKey"`
	Username string                          // <--- No gorm default fields required
	Orders   []OrderPreloadWithSpecificModel `gorm:"foreignKey:UserID;references:ID"`
}

type OrderPreloadWithSpecificModel struct {
	gorm.Model
	UserID uint `json:"user_id"`
	Price  float64
	Detail string
}

type CustomOrderRes struct {
	UserID uint `json:"user_id_custom"` // <--- Custom json key name
	Price  float64
	// Detail string <--- Notice that Detail Attribute is not required here
}

func TestPreloadWithSpecificModel(t *testing.T) {
	_ = DB.Migrator().DropTable(&UserPreloadWithSpecificModel{}, &OrderPreloadWithSpecificModel{})
	if err := DB.Migrator().AutoMigrate(&UserPreloadWithSpecificModel{}, &OrderPreloadWithSpecificModel{}); err != nil {
		t.Fatal(err)
	}
	user := UserPreloadWithSpecificModel{Username: "someone"}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.ID == 0 {
		if err := DB.Where(&user).First(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("got user: %+v", user)

	order := OrderPreloadWithSpecificModel{UserID: user.ID, Price: 999, Detail: "test order"}
	if err := DB.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	if order.ID == 0 {
		if err := DB.Where(&order).First(&order).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("got order: %+v", user)
	println("------------------------------------------------------------")

	var userCustom CustomUserRes
	var ordersCustom CustomOrderRes
	db := DB.Preload("Orders", func(tx *gorm.DB) *gorm.DB {
		return tx.Model(order).Scan(&ordersCustom) // <--- I want to fit the preload result to ordersCustom
	}).Model(&user).First(&userCustom)
	if err := db.Error; err != nil {
		t.Fatal(err)
	}
	if ordersCustom.UserID == order.UserID {
		t.Logf("got order: %+v", ordersCustom)
	} else {
		t.Errorf("got user ID is %d, want %d", ordersCustom.UserID, order.UserID)
	}
}
