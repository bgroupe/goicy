package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/bgroupe/goicy/config"
	"github.com/bgroupe/goicy/logger"
	"github.com/bgroupe/goicy/playlist"
	"github.com/gofiber/fiber/v2"
)

// MAYBE: Server container
type APIServer struct {
	app     *fiber.App
	wg      sync.WaitGroup
	counter int32
}

func NewAPIServer() *fiber.App {
	app := fiber.New()
	// do a config thing here
	// TODO: use jwt middlware
	// apiGroup := app.Group("/api")

	// register routes
	app.Get("/healthz", healthz)

	// API
	api := app.Group("/api")

	// Get current track, most useful
	api.Get("/now-playing", nowPlaying)

	// All playlist
	playlist := api.Group("/playlist")

	playlist.Get("/sessions/:session", getSessionPlaylist)
	playlist.Get("/sessions", getSessions)
	playlist.Get("/sessions-current", getSessions)
	// TODO:
	// playlist.Get("/sessions/:session/:track", getSessionTrack)
	playlist.Get("/control", getPlaylistControl)

	return app
}

func listRoutes() []string {
	return []string{
		"/healthz",
		"/api/now-playing",
		"/api/playlist/tracks/current",
		"/api/playlist/tracks/:session",
		"/api/playlist/sessions-current",
		"/api/playlist/sessions",
		"/api/playlist/sessions/:session",
		"/api/playlist/control",
	}
}

// Gets health and network info
func healthz(c *fiber.Ctx) error {
	server := c.App().Server()

	var (
		concurrency uint32
		connections int32
	)
	concurrency = server.GetCurrentConcurrency()
	connections = server.GetOpenConnectionsCount()

	err := c.JSON(fiber.Map{
		"status":             "OK",
		"active-connections": concurrency,
		"open-connections":   connections,
		"host":               server.Name,
	})

	// DONE: show connected clients
	// TODO: Show uptime: This requires a timer to be created an attached to a struct along with the API server
	if err != nil {
		logger.Logf("error checking service health, %e", 1, err.Error())
	}

	return err
}

// Now Playing
func nowPlaying(c *fiber.Ctx) error {
	db, err := createDbConn()
	if err != nil {
		logger.Log("error connecting to db", 0)
		return err
	}

	defer db.Conn.Close()

	session, err := db.GetCurrentSession()

	if err != nil {
		logger.Logf("error getting current session", 0, err.Error())
	}

	defer db.Conn.Close()

	jsonBytes, err := db.GetJsonStruct(fmt.Sprintf("%v-now-playing", session))

	if err != nil {
		logger.Logf("now-playing struct not found @%v", 1, err.Error())
	}
	// TODO: Temporary; Need to unmarshal into NowPlaying struct
	var result map[string]interface{}
	json.Unmarshal(jsonBytes, &result)

	c.JSON(result)

	return err
}

// ###################
// #### Sessions #####
// ###################

// Gets both list of all sessions from filesystem and current session
func getSessions(c *fiber.Ctx) error {
	db, err := createDbConn()
	if err != nil {
		logger.Log("error connecting to db", 0)
		return err
	}

	session, err := db.GetCurrentSession()
	sessions, err := db.GetSessions()

	if err != nil {
		logger.Log("db error querying sessions", 0)
		return err
	}

	if strings.Contains(c.Request().URI().String(), "current") {
		err = c.JSON(fiber.Map{
			"current-session": session,
		})
	} else {
		err = c.JSON(fiber.Map{
			"sessions": sessions,
		})
	}

	return err
}

func getSessionPlaylist(c *fiber.Ctx) error {
	db, err := createDbConn()
	if err != nil {
		logger.Log("error connecting to db", 0)
		return err
	}

	defer db.Conn.Close()

	sessionQuery := c.Params("session")
	if sessionQuery != "" {
		sessions, err := db.GetSessions()

		if err != nil {
			logger.Log("db error querying sessions", 0)
			return err
		} else {
			if contains(sessions, sessionQuery) {
				jsonBytes, err := db.GetJsonStruct(fmt.Sprintf("%v-playlist", sessionQuery))
				if err != nil {
					logger.Logf("Session playlist not found not found @%v", 1, err.Error())
				}
				// TODO: unmarshal into playlist struct
				var result map[string]interface{}
				json.Unmarshal(jsonBytes, &result)

				c.JSON(result)
			} else {
				return c.Status(404).SendString(errors.New("Session Not Found").Error())
			}
		}

	}
	return err
}

// Get playlist controls
func getPlaylistControl(c *fiber.Ctx) error {
	db, err := createDbConn()
	if err != nil {
		logger.Log("error connecting to db", 0)
		return err
	}

	defer db.Conn.Close()

	jsonBytes, err := db.GetJsonStruct("playlist-control")
	if err != nil {
		logger.Logf("error query playlist control  @%v", 1, err.Error())
	}

	var result map[string]interface{}
	json.Unmarshal(jsonBytes, &result)

	c.JSON(result)

	return err
}

// ##################
// #### private #####
// ##################

// Find returns the smallest index i at which x == a[i],
// or len(a) if there is no such index.
func find(a []string, s string) int {
	for i, n := range a {
		if s == n {
			return i
		}
	}
	return len(a)
}

// Contains tells whether a contains x.
func contains(a []string, s string) bool {
	for _, n := range a {
		if s == n {
			return true
		}
	}
	return false
}

func createDbConn() (playlist.DB, error) {
	db, err := playlist.ConnectDB(config.Cfg.RedisURL)
	return db, err
}

// TODO: implement channel communication for reload functionality
// 1. Send reload request with new playlist
// 2. redis-pubsub subscribes and pre-downloads in a goroutine
// 3. Next() function looks for reload key and loads new playlist from the db
// DONE: implement basic metadata fetching from redis db
//.NOTE: Need to implment app struct or benchmark
// MOAR: https://docs.gofiber.io/ctx#json
// MOAR: https://medium.com/@irshadhasmat/golang-simple-json-parsing-using-empty-interface-and-without-struct-in-go-language-e56d0e69968
// MOAR: http://liamkaufman.com/blog/2012/06/04/redis-and-relational-data/
