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

func ExtractJwtMClaims(c *fiber.Ctx) (claims jwt.MapClaims, err error) {
	user := c.Locals("user")
	if user == nil {
		return jwt.MapClaims{}, errors.New("No token found")
	}

	token := user.(*jwt.Token)
	if errors.Is(jwt.ErrTokenExpired, err) || errors.Is(jwt.ErrTokenNotValidYet, err) {
		return jwt.MapClaims{}, errors.New("Invalide token")
	}

	claims = token.Claims.(jwt.MapClaims)

	return claims, nil
}

func JwtMiddleware() fiber.Handler {
	return jwtware.New(
		jwtware.Config{
			SigningKey:  jwtware.SigningKey{Key: []byte("SuperSecretKey")},
			TokenLookup: "cookie:session",
			ContextKey:  "user",
			ErrorHandler: func(c *fiber.Ctx, err error) error {
				log.Errorw("ERROR MIDDLE", "reason", err.Error())
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Unauthorized",
				})
			},
		},
	)
}
