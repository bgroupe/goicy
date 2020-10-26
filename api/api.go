package api

import (
	"encoding/json"

	"github.com/bgroupe/goicy/config"
	"github.com/bgroupe/goicy/logger"
	"github.com/bgroupe/goicy/playlist"
	"github.com/gofiber/fiber/v2"
)

func NewAPIServer() *fiber.App {
	app := fiber.New()
	// do a config thing here
	// TODO: use jwt middlware
	// apiGroup := app.Group("/api")

	// register routes
	app.Get("/now-playing", nowPlaying)
	return app
}

func listRoutes() []string {
	return []string{
		"/api/now-playing",
		"/api/playlist",
		"/api/playlist/tracks",
		"/api/playlist/current-session",
		"/api/playlist/sessions",
		"/api/playlist-control",
	}
}

// handlers

func nowPlaying(c *fiber.Ctx) error {
	db, err := createDbConn()
	if err != nil {
		logger.Log("error connecting to db", 0)
		return err
	}

	defer db.Conn.Close()

	jsonBytes, err := db.GetJsonStruct("now-playing")

	if err != nil {
		logger.Logf("now-playing struct not found @%v", 1, err.Error())
	}
	// TODO: Temporary
	// NOTE: Unmarshal into NowPlaying struct
	var result map[string]interface{}
	json.Unmarshal(jsonBytes, &result)

	c.JSON(result)
	// MAYBE: use context as struct key...
	// logger.Logf("here is the URI: %v", 1, c.Context().Request.URI().String())

	return err
}

func createDbConn() (playlist.DB, error) {
	db, err := playlist.ConnectDB(config.Cfg.RedisURL)
	return db, err
}

// TODO: implement channel communication for reload functionality
// 1. Send reload request with new playlist
// 2. redis-pubsub subscribes and pre-downloads in a goroutine
// 3. Next() function looks for reload key and loads new playlist from the db
// TODO: implement basic metadata fetching from redis db
//.NOTE: Need to implment app struct or benchmark
// MOAR: https://docs.gofiber.io/ctx#json
// MOAR: https://medium.com/@irshadhasmat/golang-simple-json-parsing-using-empty-interface-and-without-struct-in-go-language-e56d0e69968
// MOAR: http://liamkaufman.com/blog/2012/06/04/redis-and-relational-data/
