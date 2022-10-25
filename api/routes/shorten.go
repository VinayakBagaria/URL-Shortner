package routes

import (
	"github.com/VinayakBagaria/url-shortner/database"
	"github.com/VinayakBagaria/url-shortner/helpers"
	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"os"
	"strconv"
	"time"
)

type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL             string        `json:"url"`
	CustomShort     string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	XRateRemaining  int           `json:"rate_limit"`
	XRateLimitReset time.Duration `json:"rate_limit_reset"`
}

func ShortenURL(c *fiber.Ctx) error {
	body := &request{}
	if err := c.BodyParser(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	// implement rate limiting
	r2 := database.CreateClient(1)
	defer r2.Close()
	val, err := r2.Get(c.IP()).Result()
	if err == redis.Nil {
		r2.Set(c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := r2.TTL(c.IP()).Result()
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":            "rate limit exceeded",
				"rate_limit_reset": limit / time.Nanosecond / time.Minute,
			})
		}
	}

	// check if url is valid
	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid URL"})
	}

	// check for domain error
	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "you can't access this request"})
	}

	// enforce https, ssl
	body.URL = helpers.EnforceHTTP(body.URL)

	var id string
	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	r := database.CreateClient(0)
	defer r.Close()
	val, _ = r.Get(id).Result()
	if val != "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "URL custom short is already in use"})
	}
	if body.Expiry == 0 {
		body.Expiry = 24
	}

	err = r.Set(id, body.URL, body.Expiry*3600*time.Second).Err()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "unable to connect to server"})
	}

	r2.Decr(c.IP())
	val, _ = r2.Get(c.IP()).Result()
	rateRemaining, _ := strconv.Atoi(val)
	ttl, _ := r2.TTL(c.IP()).Result()

	return c.Status(fiber.StatusOK).JSON(response{
		URL:             body.URL,
		CustomShort:     os.Getenv("DOMAIN") + "/" + id,
		Expiry:          body.Expiry,
		XRateRemaining:  rateRemaining,
		XRateLimitReset: ttl / time.Nanosecond / time.Minute,
	})
}
