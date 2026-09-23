package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/config"
)

// withTempConfig 切换工作目录到一个临时目录，避免污染。
func withTempConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prev, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prev) })
	return dir
}

func writeYAML(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}

func TestLoad_ReadsServiceYAML(t *testing.T) {
	// Arrange
	dir := withTempConfig(t)
	writeYAML(t, filepath.Join(dir, "config", "auth.yaml"), `
service: auth
http:
  addr: ":8080"
db:
  dsn: "postgres://u:p@localhost:5432/d"
  max_conns: 10
redis:
  addr: "localhost:6379"
kafka:
  brokers: ["localhost:9092"]
auth:
  jwt_secret: "test-secret"
  jwt_ttl: "24h"
`)

	// Act
	cfg, err := config.Load("auth")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "auth", cfg.Service)
	assert.Equal(t, ":8080", cfg.HTTP.Addr)
	assert.Equal(t, "postgres://u:p@localhost:5432/d", cfg.DB.DSN)
	assert.Equal(t, int32(10), cfg.DB.MaxConns)
	assert.Equal(t, "localhost:6379", cfg.Redis.Addr)
	assert.Equal(t, []string{"localhost:9092"}, cfg.Kafka.Brokers)
	assert.Equal(t, "test-secret", cfg.Auth.JWTSecret)
}

func TestLoad_EnvOverridesYAML(t *testing.T) {
	// Arrange
	dir := withTempConfig(t)
	writeYAML(t, filepath.Join(dir, "config", "auth.yaml"), `
service: auth
http:
  addr: ":8080"
auth:
  jwt_secret: "from-yaml"
  jwt_ttl: "24h"
`)
	t.Setenv("DOCTORS_AUTH_JWT_SECRET", "from-env")
	t.Setenv("DOCTORS_HTTP_ADDR", ":9999")

	// Act
	cfg, err := config.Load("auth")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "from-env", cfg.Auth.JWTSecret)
	assert.Equal(t, ":9999", cfg.HTTP.Addr)
}

func TestLoad_MissingFileReturnsError(t *testing.T) {
	// Arrange
	withTempConfig(t)
	// 不写 yaml

	// Act
	_, err := config.Load("ghost")

	// Assert
	assert.Error(t, err)
}

func TestMustLoad_PanicsOnError(t *testing.T) {
	// Arrange
	withTempConfig(t)

	// Act & Assert
	assert.Panics(t, func() {
		config.MustLoad("ghost")
	})
}

func TestHTTPConfig_Defaults(t *testing.T) {
	// Arrange
	withTempConfig(t)
	writeYAML(t, filepath.Join("config", "auth.yaml"), `
service: auth
http: {}
auth:
  jwt_secret: "x"
  jwt_ttl: "1h"
`)

	// Act
	cfg, err := config.Load("auth")
	require.NoError(t, err)

	// Assert
	assert.Equal(t, ":8080", cfg.HTTP.Addr, "默认值")
	assert.Equal(t, 30, cfg.HTTP.ReadTimeoutSec)
}