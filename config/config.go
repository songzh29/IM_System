package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Mysql  MysqlConfig  `mapstructure:"mysql"`
	Redis  RedisConfig  `mapstructure:"redis"`
	JWT    JWTConfig    `mapstructure:"jwt"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type MysqlConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire"`
}

var ConfigInfo *Config // 此时 ConfigInfo = nil

func Init() error {

	viper.SetConfigName("config")       // 文件名（不带扩展名）
	viper.SetConfigType("yaml")         // 类型
	viper.AddConfigPath(".")            // 查找路径
	viper.AddConfigPath("../../config") // 查找路径

	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}
	ConfigInfo = &Config{}
	err = viper.Unmarshal(ConfigInfo)
	if err != nil {
		return fmt.Errorf("反序列化失败: %w", err)
	}
	return nil

}
