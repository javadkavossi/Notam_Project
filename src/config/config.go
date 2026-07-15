package config

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Password PasswordConfig
	Cors     CorsConfig
	Logger   LoggerConfig
	Otp      OtpConfig
	JWT      JWTConfig
	Solace   SolaceConfig
	Auth     AuthConfig
}

// SolaceConfig تنظیمات اتصال به منبع NOTAM (FAA SWIM).
// مقادیر حساس (Username/Password) فقط از متغیر محیطی خوانده می‌شوند و هیچ مقدار پیش‌فرضی در کد ندارند.
type SolaceConfig struct {
	Host     string
	VPN      string
	Username string
	Password string
	Queue    string
}

// AuthConfig احراز هویت موقت (تا پیاده‌سازی JWT در E0-2/E0-3).
// از متغیر محیطی خوانده می‌شود؛ بدون مقدار پیش‌فرض محرمانه در کد.
type AuthConfig struct {
	User string
	Pass string
}

type ServerConfig struct {
	InternalPort string
	ExternalPort string
	RunMode      string
}

type LoggerConfig struct {
	FilePath string
	Encoding string
	Level    string
	Logger   string
}

type PostgresConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DbName          string
	SSLMode         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Host               string
	Port               string
	Password           string
	Db                 string
	DialTimeout        time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleCheckFrequency time.Duration
	PoolSize           int
	PoolTimeout        time.Duration
}

type PasswordConfig struct {
	IncludeChars     bool
	IncludeDigits    bool
	MinLength        int
	MaxLength        int
	IncludeUppercase bool
	IncludeLowercase bool
}

type CorsConfig struct {
	AllowOrigins string
}

type OtpConfig struct {
	ExpireTime time.Duration
	Digits     int
	Limiter    time.Duration
}

type JWTConfig struct {
	AccessTokenExpireDuration  time.Duration
	RefreshTokenExpireDuration time.Duration
	Secret                     string
	RefreshSecret              string
}

func GetConfig() *Config {
	// دیباگ اطلاعات محیط
	wd, _ := os.Getwd()

	log.Printf("Current working directory: %s", wd)

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	log.Printf("APP_ENV: %s", env)

	cfgPath := getConfigPath(env)
	log.Printf("Config path: %s", cfgPath)

	v, err := LoadConfig(cfgPath, "yml")
	if err != nil {
		log.Fatalf("Error in load config %v", err)
	}

	cfg, err := ParseConfig(v)
	if err != nil {
		log.Fatalf("Error in parse config %v", err)
	}

	envPort := os.Getenv("PORT")
	if envPort != "" {
		cfg.Server.ExternalPort = envPort
		log.Printf("Set external port from environment -> %s", cfg.Server.ExternalPort)
	} else {
		cfg.Server.ExternalPort = cfg.Server.InternalPort
		log.Printf("Set external port from internal port -> %s", cfg.Server.ExternalPort)
	}

	applyEnvOverrides(cfg)

	return cfg
}

// applyEnvOverrides مقادیر حساس و متغیرهای محیطی را روی config اعمال می‌کند.
// اصل امنیتی (E0-1): هیچ رمز یا نام‌کاربری در کد یا فایل config کامیت نمی‌شود؛
// این مقادیر فقط از متغیر محیطی می‌آیند. مقادیر غیرحساس (host/vpn/queue) پیش‌فرض دارند.
func applyEnvOverrides(cfg *Config) {
	// ---- Solace (منبع NOTAM) ----
	cfg.Solace.Host = getEnv("SOLACE_HOST", firstNonEmpty(cfg.Solace.Host, "tcps://ems2.swim.faa.gov:55443"))
	cfg.Solace.VPN = getEnv("SOLACE_VPN", firstNonEmpty(cfg.Solace.VPN, "AIM_FNS"))
	cfg.Solace.Queue = getEnv("SOLACE_QUEUE", cfg.Solace.Queue)
	cfg.Solace.Username = getEnv("SOLACE_USERNAME", cfg.Solace.Username) // بدون پیش‌فرض؛ فقط از env
	cfg.Solace.Password = getEnv("SOLACE_PASSWORD", cfg.Solace.Password) // بدون پیش‌فرض؛ فقط از env

	// ---- Postgres/Redis: اجازهٔ override رمز از env (تا رمز در فایل config نماند) ----
	cfg.Postgres.Password = getEnv("POSTGRES_PASSWORD", cfg.Postgres.Password)
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", cfg.Redis.Password)

	// ---- Auth موقت ----
	cfg.Auth.User = getEnv("AUTH_USER", firstNonEmpty(cfg.Auth.User, "admin"))
	cfg.Auth.Pass = getEnv("AUTH_PASS", cfg.Auth.Pass) // بدون پیش‌فرض محرمانه

	// ---- JWT secret از env ----
	cfg.JWT.Secret = getEnv("JWT_SECRET", cfg.JWT.Secret)
	cfg.JWT.RefreshSecret = getEnv("JWT_REFRESH_SECRET", cfg.JWT.RefreshSecret)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func ParseConfig(v *viper.Viper) (*Config, error) {
	var cfg Config
	err := v.Unmarshal(&cfg)
	if err != nil {
		log.Printf("Unable to parse config: %v", err)
		return nil, err
	}
	return &cfg, nil
}

func LoadConfig(filename string, fileType string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigType(fileType)

	// در Docker مسیر صریح استفاده شود
	if os.Getenv("APP_ENV") == "docker" {
		explicitPath := "/app/config/" + filename + "." + fileType
		v.SetConfigFile(explicitPath)
	} else {
		v.SetConfigName(filename)
		// مسیرهای جستجو برای فایل config
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("./src")
		v.AddConfigPath("./src/config")
		v.AddConfigPath("/app/config")
		v.AddConfigPath("../src/config")
		v.AddConfigPath("../../src/config")
	}

	v.AutomaticEnv()

	err := v.ReadInConfig()
	if err != nil {
		log.Printf("Unable to read config: %v", err)
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, errors.New("config file not found")
		}
		return nil, err
	}

	log.Printf("Config file found: %s", v.ConfigFileUsed())
	return v, nil
}

func getConfigPath(env string) string {
	if env == "docker" {
		return "config-docker"
	} else if env == "production" {
		return "config-production"
	} else {
		return "config-development"
	}
}
