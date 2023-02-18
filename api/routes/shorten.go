package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/dacore-x/url-shortener/database"
	"github.com/dacore-x/url-shortener/helpers"

	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short_url"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL             string        `json:"url"`
	CustomShort     string        `json:"short_url"`
	Expiry          time.Duration `json:"expiry"`
	XRateRemaining  int           `json:"rate_limit"`
	XRateLimitReset time.Duration `json:"rate_limit_reset"`
}

// ShortenURL return shorten URL for user request
func ShortenURL(c *fiber.Ctx) error {
	// parsing the body of user's request
	body := new(request)
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "error while parsing the URL"})
	}

	// implementation of request limiting by api quota
	// connecting to redis {"USER_IP": "REQUESTS_PER_PERIOD"} db
	userIPClient := database.CreateClient(1) 
	defer userIPClient.Close()

	// getting user's amount of requests from redis by ip of request
	value, err := userIPClient.Get(database.Ctx, c.IP()).Result()
	if err == redis.Nil {
		// expiration is 30 minutes = 30 * 60 * time.Second
		_ = userIPClient.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		valInt, _ := strconv.Atoi(value)

		// checking for request limit exceed
		if valInt <= 0 {
			// getting remaining TTL of user's IP key
			ttl, _ := userIPClient.TTL(database.Ctx, c.IP()).Result()
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":            "Rate limit exceeded",
				"rate_limit_reset": ttl / time.Nanosecond / time.Minute, // converting to minutes
			})

		}
	}

	// checking for correct URL format
	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid URL"})
	}

	// checking for domain error
	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "domain error"})
	}

	// enforce https, SSL
	body.URL = helpers.EnforceHTTP(body.URL)

	var id string

	// setting custom short url for user's url
	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	// connecting to redis {"URL": "Short URL"} db
	shortenURLClient := database.CreateClient(0)
	defer shortenURLClient.Close()

	// checking for existence of short url in db
	val, _ := shortenURLClient.Get(database.Ctx, id).Result()
	if val != "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "custom short URL is already in use",
		})
	}

	// setting expiration of url in days
	if body.Expiry == 0 {
		body.Expiry = 10
	}

	// setting expiration in redis
	// expiration is body.Expiry(days)*24(hours)
	err = shortenURLClient.Set(database.Ctx, id, body.URL, body.Expiry*24*time.Hour).Err()

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "unable to connect to server",
		})
	}

	resp := response{
		URL:             body.URL,
		CustomShort:     "",
		Expiry:          body.Expiry,
		XRateRemaining:  10,
		XRateLimitReset: 30,
	}

	// decrementing user's shorten url requests remaining amount before reset
	userIPClient.Decr(database.Ctx, c.IP())

	// getting remaining amount of user's request
	val, _ = userIPClient.Get(database.Ctx, c.IP()).Result()
	resp.XRateRemaining, _ = strconv.Atoi(val)

	// getting remaining TTL of user's IP key
	ttl, _ := userIPClient.TTL(database.Ctx, c.IP()).Result()
	resp.XRateLimitReset = ttl / time.Nanosecond / time.Minute
	
	// setting request response shorten url
	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id

	// returning response to user
	return c.Status(fiber.StatusOK).JSON(resp)
}
