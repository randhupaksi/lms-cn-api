package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"lms-cn-api/internal/config"
	"lms-cn-api/internal/database"
	"lms-cn-api/internal/modules/users"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Open(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	identifier := strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_IDENTIFIER")))
	fullName := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_NAME"))
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	role := users.Role(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_ROLE")))
	if role == "" {
		role = users.RoleAdmin
	}
	if role != users.RoleAdmin && role != users.RoleSuperadmin {
		log.Fatal("BOOTSTRAP_ADMIN_ROLE must be admin or superadmin")
	}
	if identifier == "" || fullName == "" || len(password) < 12 {
		log.Fatal("BOOTSTRAP_ADMIN_IDENTIFIER, BOOTSTRAP_ADMIN_NAME, and a password of at least 12 characters are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	repository := users.NewRepository(db.GORM)
	if role == users.RoleSuperadmin {
		var count int64
		if err := db.GORM.Model(&users.User{}).Where("role=?", role).Count(&count).Error; err != nil {
			log.Fatal("cannot inspect bootstrap state")
		}
		if count > 0 {
			log.Fatal("a superadmin already exists; manage additional superadmins through access governance")
		}
	}
	if _, err := repository.FindByIdentifier(ctx, identifier); err == nil {
		log.Fatal("an account with the bootstrap identifier already exists")
	} else if !errors.Is(err, users.ErrNotFound) {
		log.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	admin := users.User{ID: uuid.NewString(), Identifier: identifier, FullName: fullName, Role: role, Status: users.StatusActive, PasswordHash: string(hash), MustChangePassword: true}
	if err := repository.Create(ctx, &admin); err != nil {
		log.Fatal(err)
	}
	log.Print("administrator created successfully")
}
