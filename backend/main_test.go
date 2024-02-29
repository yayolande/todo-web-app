package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestUserIsValid(t *testing.T) {
	var tests = []struct {
		input User
		want  error
	}{
		{input: User{Id: 0}, want: ErrEmptyUsernamePassword},
		{input: User{Id: -2}, want: ErrEmptyUsernamePassword},
		{input: User{Id: 2}, want: ErrUserAlreadyExist},
		{input: User{Id: 2, Username: "Melo", Password: "Password"}, want: ErrUserAlreadyExist},
		{input: User{Id: 0, Username: "", Password: ""}, want: ErrEmptyUsernamePassword},
		{input: User{Id: 0, Username: "Melo", Password: "Melo"}, want: nil},
		{input: User{Id: 4, Username: "Melo", Password: "Melo"}, want: ErrUserAlreadyExist},
	}

	for _, tt := range tests {
		name := fmt.Sprint(tt.input, " ::: ", tt.want)

		t.Run(name, func(t *testing.T) {
			answer := tt.input.IsValid()

			if !errors.Is(answer, tt.want) {
				t.Error("Input --> ", tt.input, " ::: want --> ", tt.want, " :::: got ---> ", answer)
			}
		})
	}
}

func TestJwtMiddlewareProtect(t *testing.T) {
	var tests = []struct {
		HeaderKey     string
		HeaderContent string
		want          int
	}{
		{HeaderKey: "", HeaderContent: "", want: fiber.StatusUnauthorized},
		{HeaderKey: "Authorization", HeaderContent: "", want: fiber.StatusUnauthorized},
		{HeaderKey: "Authorization", HeaderContent: "", want: fiber.StatusUnauthorized},
		{HeaderKey: "Authorization", HeaderContent: "LINDELOF", want: fiber.StatusUnauthorized},
		{HeaderKey: "Authorization", HeaderContent: "BASIC", want: fiber.StatusUnauthorized},
		{HeaderKey: "Authorization", HeaderContent: "BEARER", want: fiber.StatusUnauthorized},
		{HeaderKey: "Authorization", HeaderContent: "BEARER BEARER", want: fiber.StatusUnauthorized},
		{HeaderKey: "Authorization", HeaderContent: "BEARER eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c", want: fiber.StatusUnauthorized},
		{HeaderKey: "Authorization", HeaderContent: "BEARER eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMzd9.7L4RPd-LtjzTMQDrVUM6VDs1ghmcp1BDyjnQcO0oEP4", want: fiber.StatusOK},
		{HeaderKey: "Authorization", HeaderContent: "BEARER eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.t-IDcSemACt8x4iTMCda8Yhe3iZaWbvV5XKSTbuAn0M", want: fiber.StatusOK},
	}

	app := fiber.New()
	app.Use(jwtMiddlewareProtect)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": "Hello, World !",
		})
	})

	var url = "/"

	for _, tt := range tests {
		var name string = fmt.Sprintf("Header --> %v: %v <---> success: %v", tt.HeaderKey, tt.HeaderContent, tt.want)
		t.Run(name, func(t *testing.T) {
			var req *http.Request = httptest.NewRequest(http.MethodGet, url, nil)
			req.Header.Set(tt.HeaderKey, tt.HeaderContent)

			res, _ := app.Test(req)

			var body, _ = io.ReadAll(res.Body)
			fmt.Println("\nresponse body: ", string(body))

			if res.StatusCode != tt.want {
				var body, _ = io.ReadAll(res.Body)

				t.Errorf("Expected status code : %v , but got : %v ---> Body: %v ::: res : %#v", tt.want, res.StatusCode, string(body), res)
			}
		})
	}

}
