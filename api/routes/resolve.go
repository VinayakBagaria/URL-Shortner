package routes

import (
	"github.com/VinayakBagaria/url-shortner/database"
	"github.com/go-redis/redis"
	"github.com/gofiber/fiber/v2"
)

func ResolveURL(c *fiber.Ctx) error {
	url := c.Params("url")

	r := database.CreateClient(0)
	defer r.Close()

	value, err := r.Get(url).Result()
	if err == redis.Nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short not found in db"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot connect to db"})
	}

	rInr := database.CreateClient(1)
	defer rInr.Close()

	rInr.Incr("counter")
	c.Redirect(value, 301)
	return nil
}
