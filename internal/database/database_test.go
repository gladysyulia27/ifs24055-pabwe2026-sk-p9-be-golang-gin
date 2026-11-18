package database

import (
	"app/internal/config"
	"strings"
	"testing"
)

func TestConnectAndSeed(t *testing.T) {
	// 1. Connect with sqlite via Connect(cfg)
	cfg := &config.Config{
		DBConnection:  "sqlite",
		DBDatabase:    ":memory:",
		AdminUsername: "superadmin",
		AdminPassword: "secretpassword",
	}
	db, err := Connect(cfg)
	if err != nil {
		t.Fatalf("expected sqlite connection to succeed, got %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// 2. Seed new admin user
	if err := Seed(db, cfg); err != nil {
		t.Fatalf("first seed failed: %v", err)
	}

	var u User
	if err := db.Where("username = ?", "superadmin").First(&u).Error; err != nil {
		t.Fatalf("seeded admin not found: %v", err)
	}
	if u.Role != "admin" {
		t.Fatalf("expected role admin, got %s", u.Role)
	}

	// 3. Seed idempotency (second call returns nil when admin already exists)
	if err := Seed(db, cfg); err != nil {
		t.Fatalf("second seed should be idempotent, got %v", err)
	}

	// 4. Seed error when password exceeds 72 bytes (bcrypt limit)
	longPassCfg := &config.Config{
		AdminUsername: "anotheradmin",
		AdminPassword: strings.Repeat("a", 80),
	}
	if err := Seed(db, longPassCfg); err == nil {
		t.Fatal("expected error when seeding password > 72 bytes")
	}

	// 5. ConnectTest helper
	testDB, err := ConnectTest()
	if err != nil || testDB == nil {
		t.Fatalf("ConnectTest failed: %v", err)
	}

	// 6. Invalid postgres connection error path
	badCfg := &config.Config{
		DBConnection: "pgsql",
		DBHost:       "127.0.0.1",
		DBPort:       "1",
		DBUsername:   "invalid",
		DBPassword:   "invalid",
		DBDatabase:   "invalid",
		DBSSLMode:    "disable",
	}
	_, err = Connect(badCfg)
	if err == nil {
		t.Fatal("expected error connecting to invalid postgres port")
	}
}
