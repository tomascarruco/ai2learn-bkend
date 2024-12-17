package authentication

import (
	"errors"
	"time"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
)

var ErrFailedToCreateUserJWT = errors.New("could not create a user token")

// CreateSessionJwt creates a new authentication session
func CreateSessionJwt(user string) (token string, err error) {
	claims := jwt.MapClaims{
		"name": user,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	}
	userToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tk, err := userToken.SignedString([]byte("SuperSecretKey"))
	if err != nil {
		log.Debugw("Failed to create the user token", "user", user, "reason", err.Error())
		return "", ErrFailedToCreateUserJWT
	}

	return tk, nil
}

func ExtractJwtMClaims(c *fiber.Ctx) (claims jwt.MapClaims) {
	user := c.Locals("user").(*jwt.Token)
	claims = user.Claims.(jwt.MapClaims)

	return claims
}

func JwtMiddleware() fiber.Handler {
	return jwtware.New(
		jwtware.Config{
			SigningKey: jwtware.SigningKey{Key: []byte("SuperSecretKey")},
			AuthScheme: "Bearer",
		},
	)
}
