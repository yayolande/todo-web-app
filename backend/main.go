package main

import (
	"errors"
	// "fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (u User) IsEmpty() bool {
	empty := false

	if u.Id <= 0 {
		empty = true
	}

	return empty
}

func (user User) IsValid() error {
	if !user.IsEmpty() {
		return ErrUserAlreadyExist
	}

	if len(strings.Trim(user.Username, " ")) <= 0 ||
		len(strings.Trim(user.Password, " ")) <= 0 {
		return ErrEmptyUsernamePassword
	}

	return nil
}

const MSG_NOT_IMPLEMENTED string = "Not Implemented"
const jwtSigningKey = "secret"

var (
	gormDB *gorm.DB
)

var (
	ErrUserAlreadyExist      = errors.New("User Already Exist !")
	ErrEmptyUsernamePassword = errors.New("Username or Password empty !")
)

func main() {
	gormDB = openDB()

	app := fiber.New()
	setupRoute(app)

	port := ":2200"
	app.Listen(port)
}

func openDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("to_do.db"), &gorm.Config{})

	if err != nil {
		log.Println("Unable to open the database, err = ", err.Error())
		os.Exit(22)
	}

	db.AutoMigrate(&User{})

	return db
}

func setupRoute(app *fiber.App) {
	app.Static("/", "../")

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Hello World !",
		})
	})

	api := app.Group("/api/v1")

	api.Post("/register", func(c *fiber.Ctx) error {
		user := User{}

		if err := c.BodyParser(&user); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		err := gormDB.Limit(1).Find(&user).Error
		if err != nil {
			log.Println(c.Route().Name, " --> ", err.Error())
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		err = user.IsValid()
		if err != nil {
			log.Println(c.Route().Name, " --> ", err.Error())
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		err = gormDB.Create(&user).Error
		log.Print("user = ", user)
		if err != nil {
			log.Println(c.Route().Name, " --> ", err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"user": user,
		})
	})

	api.Post("/login", func(c *fiber.Ctx) error {
		user := User{}

		if err := c.BodyParser(&user); err != nil {
			log.Println(c.Route().Path, " --> ", err.Error())
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		foundUser := User{}
		err := gormDB.Where("username = ? AND password = ?", user.Username, user.Password).First(&foundUser).Error

		if err != nil {
			log.Println(c.Route().Path, " --> ", err.Error())
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		claims := jwt.MapClaims{
			"user_id": foundUser.Id,
		}

		key := []byte(jwtSigningKey)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString(key)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"token": tokenString,
		})
	})

	api.Use(jwtMiddlewareProtect)
	api.Get("/empty", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "OK",
		})
	})
}

func jwtMiddlewareProtect(c *fiber.Ctx) error {
	// Extract Token from header
	auths := c.GetReqHeaders()["Authorization"]

	if len(auths) < 1 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "No Authorization Header Available",
		})
		// return errors.New("No Authorization Header Available")
	}

	authHeader := auths[0]
	authTokens := strings.Split(authHeader, "BEARER")

	if len(authTokens) != 2 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Authorization header malformated around 'BEARER'",
		})
	}

	tokenString := strings.Trim(authTokens[1], " ")

	// Validate the token
	claims := &struct {
		jwt.RegisteredClaims
		UserId int
	}{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSigningKey), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	c.Locals("user_id", claims.UserId)

	return c.Next()
}
