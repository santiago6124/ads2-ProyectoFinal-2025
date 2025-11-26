package main

import (
	"log"

	"users-api/internal/models"
	"users-api/pkg/database"
	"users-api/pkg/utils"
)

func main() {
	// Connect to database
	db, err := database.NewConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to database successfully")

	// Run migrations first
	if err := db.AutoMigrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations completed successfully")

	// Seed admin user
	if err := seedAdminUser(db); err != nil {
		log.Fatalf("Failed to seed admin user: %v", err)
	}

	log.Println("Database seeding completed successfully")
}

func seedAdminUser(db *database.DB) error {
	// Check if admin user already exists
	var existingUser models.User
	result := db.DB.Where("email = ?", "admin@cryptosim.com").First(&existingUser)

	if result.Error == nil {
		log.Println("Admin user already exists, skipping...")
		return nil
	}

	// Hash admin password
	hashedPassword, err := utils.HashPassword("admin1234")
	if err != nil {
		return err
	}

	// Create admin user
	adminUser := models.User{
		Username:       "admin",
		Email:          "admin@cryptosim.com",
		PasswordHash:   hashedPassword,
		Role:           models.RoleAdmin,
		InitialBalance: 100000.00,
		IsActive:       true,
	}

	// Save to database
	if err := db.DB.Create(&adminUser).Error; err != nil {
		return err
	}

	log.Printf("Admin user created successfully (ID: %d)", adminUser.ID)
	return nil
}
