package playlist

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/nitishm/go-rejson"
)

type DB struct {
	Conn        redis.Conn
	JsonHandler *rejson.Handler
}

// SET One Object
func (db *DB) AddJsonStruct(key string, value interface{}) (res interface{}, err error) {
	res, err = db.JsonHandler.JSONSet(key, ".", value)
	if err != nil {
		fmt.Println(err)
	}
	return res, err
}

// GET One Object
func (db *DB) GetJsonStruct(key string) (json []byte, err error) {
	res, err := db.JsonHandler.JSONGet(key, ".")
	if err != nil {
		fmt.Println(err)
	}
	json, err = redis.Bytes(res, err)
	if err != nil {
		fmt.Println(err)
	}
	return json, err
}

// Update a value with json path
// example `JSON.SET foo .bar.baz `"{\"thing\": \"false\"}"`
func (db *DB) UpdateJsonStruct(key string, path string, value interface{}) (res interface{}, err error) {
	res, err = db.JsonHandler.JSONSet(key, path, value)
	if err != nil {
		fmt.Println(err)
	}
	return res, err
}

// Get value at path
func (db *DB) GetJsonStructPath(key string, path string) (res interface{}, err error) {
	res, err = db.JsonHandler.JSONGet(key, path)
	if err != nil {
		fmt.Println(err)
	}
	json, err := redis.Bytes(res, err)
	if err != nil {
		fmt.Println(err)
	}
	return json, err
}

// Appends session to Redis list of sessions. If appended successfully, `current-session key will be set with the same value`
func (db *DB) AppendListSession(session string) (res interface{}, err error) {
	res, err = db.Conn.Do("LPUSH", "sessions", session)

	if err != nil {
		fmt.Println(err)
	}

	_, err = db.Conn.Do("SET", "current-session", session)

	if err != nil {
		fmt.Println(err)
	}

	return res, err
}

// Gets list of Sessions. Returns array of strings
func (db *DB) GetSessions() (res []string, err error) {
	res, err = redis.Strings(db.Conn.Do("LRANGE", "sessions", "0", "-1"))
	if err != nil {
		fmt.Println(err)
	}

	return res, err
}

// Gets list of Sessions. Returns array of strings
func (db *DB) GetCurrentSession() (res string, err error) {
	res, err = redis.String(db.Conn.Do("GET", "current-session"))
	if err != nil {
		fmt.Println(err)
	}

	return res, err
}

// Constructor
func ConnectDB(url string) (db DB, err error) {
	conn, err := redis.DialURL(url)
	if err != nil {
		fmt.Println("error connecting to db")
	}

	rh := rejson.NewReJSONHandler()
	rh.SetRedigoClient(conn)

	db = DB{
		Conn:        conn,
		JsonHandler: rh,
	}

	return db, err
}
