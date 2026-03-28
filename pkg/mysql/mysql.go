package mysqldb

import (
	"fmt"
	"time"

	"github.com/songzh29/IM_System/config"
	"github.com/songzh29/IM_System/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() error {
	host := config.ConfigInfo.Mysql.Host
	dbname := config.ConfigInfo.Mysql.DBName
	password := config.ConfigInfo.Mysql.Password
	port := config.ConfigInfo.Mysql.Port
	user := config.ConfigInfo.Mysql.User
	maxOpenConns := config.ConfigInfo.Mysql.MaxOpenConns
	maxIdleConns := config.ConfigInfo.Mysql.MaxIdleConns

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, dbname)
	var err error
	DB, err = gorm.Open(mysql.Open(dsn))
	if err != nil {
		return err
	}
	DB.AutoMigrate(&model.User{})
	sqlDB, _ := DB.DB()
	sqlDB.SetMaxOpenConns(maxOpenConns)        // 最大连接数
	sqlDB.SetMaxIdleConns(maxIdleConns)        // 最大空闲连接
	sqlDB.SetConnMaxLifetime(time.Hour)        // 连接最大生命周期
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) //连接多久没使用会被kill

	return nil
}
