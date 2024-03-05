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

type Todo struct {
	Id          int    `json:"id"`
	Title       string `json:"title"`
	DateEpoch   int    `json:"date_epoch"`
	IsCompleted bool   `json:"is_completed"`
	IsDeleted   bool   `json:"is_deleted"`
	UserId      int    `json:"user_id"`
	User        User   `json:"user" gorm:"foreignKey:UserId"`
}

type User struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
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
	// db.AutoMigrate(&Todo{})

	return db
}

func setupRoute(app *fiber.App) {

	app.Use(func(c *fiber.Ctx) error {
		log.Println("Hello From CORS policy manager handler !")

		method := c.Route().Method
		log.Println("Method: ", method)
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Set("Access-Control-Allow-Credentials", "true")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")

		if method == "OPTIONS" {
			log.Println("OPTIONS is OK")
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	})

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

	api.Get("/todo", func(c *fiber.Ctx) error {
		var userId int = getUserIdFromMiddlewareContext(c)

		todos := []Todo{}
		err := gormDB.Where("user_id = ?", userId).Find(&todos).Error

		if err != nil {
			log.Println(c.Route().Path, " ==> ", err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"todos": todos,
		})
	})

	api.Post("/todo", func(c *fiber.Ctx) error {
		inboundTodo := Todo{}

		if err := c.BodyParser(&inboundTodo); err != nil {
			log.Println(c.Route().Path, " --> ", err.Error())
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		inboundTodo.UserId = getUserIdFromMiddlewareContext(c)
		err := gormDB.Create(&inboundTodo).Error

		if err != nil {
			log.Println(c.Route().Path, " --> ", err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"todo": inboundTodo,
		})
	})

	api.Delete("/todo/:id<int>", func(c *fiber.Ctx) error {
		userId := getUserIdFromMiddlewareContext(c)
		todoId := c.Params("id", "-1")

		// err := gormDB.Delete(&Todo{}, todoId).Error
		err := gormDB.
			Where("user_id = ? AND id = ?", userId, todoId).
			Delete(&Todo{}).
			Error

		if err != nil {
			log.Println(c.Route().Path, " --> ", err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"id_deleted": todoId,
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
		UserId int `json:"user_id"`
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

func getUserIdFromMiddlewareContext(c *fiber.Ctx) int {
	var userId int = c.Locals("user_id").(int)

	return userId
}
