package config

import (
	"log"
	"sync"

	"github.com/spf13/viper"
)

// Config 全局配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	GitLab   GitLabConfig   `mapstructure:"gitlab"`
	Database DatabaseConfig `mapstructure:"database"`
	Schedule ScheduleConfig `mapstructure:"schedule"`
	Filter   FilterConfig   `mapstructure:"filter"`
	AI       AIConfig       `mapstructure:"ai"`
	Projects []ProjectItem  `mapstructure:"projects"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// GitLabConfig GitLab配置
type GitLabConfig struct {
	URL        string `mapstructure:"url"`
	Token      string `mapstructure:"token"`
	APIVersion string `mapstructure:"api_version"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// ScheduleConfig 定时任务配置
type ScheduleConfig struct {
	Sync    string `mapstructure:"sync"`
	Daily   string `mapstructure:"daily"`
	Weekly  string `mapstructure:"weekly"`
	Monthly string `mapstructure:"monthly"`
}

// FilterConfig 过滤规则配置
type FilterConfig struct {
	ExcludePaths      []string `mapstructure:"exclude_paths"`
	ExcludeExtensions []string `mapstructure:"exclude_extensions"`
	ExcludeFilenames  []string `mapstructure:"exclude_filenames"`
	ExcludeMessages   []string `mapstructure:"exclude_messages"`
}

// AIConfig AI审核配置
type AIConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	BaseURL     string  `mapstructure:"base_url"`
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
	Timeout     int     `mapstructure:"timeout"`
}

// ProjectItem 项目配置项
type ProjectItem struct {
	ID      int    `mapstructure:"id"`
	Name    string `mapstructure:"name"`
	Enabled bool   `mapstructure:"enabled"`
}

var (
	globalConfig *Config
	once         sync.Once
)

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	var err error
	once.Do(func() {
		viper.SetConfigFile(configPath)
		viper.SetConfigType("yaml")

		if err = viper.ReadInConfig(); err != nil {
			log.Printf("读取配置文件失败: %v", err)
			return
		}

		globalConfig = &Config{}
		if err = viper.Unmarshal(globalConfig); err != nil {
			log.Printf("解析配置文件失败: %v", err)
			return
		}

		log.Printf("配置加载成功: 服务器端口=%d, GitLab地址=%s",
			globalConfig.Server.Port, globalConfig.GitLab.URL)
	})

	return globalConfig, err
}

// Get 获取全局配置
func Get() *Config {
	return globalConfig
}

// Reload 重新加载配置
func Reload() (*Config, error) {
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	globalConfig = cfg
	return cfg, nil
}
