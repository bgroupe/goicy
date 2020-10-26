package playlist

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
)

const (
	redisAddr = "redis://localhost:6379"
)

var jsonData = []byte(`
{
   "name": "test-plist",
   "Tracks": [
	 {
	   "title": "unsaved-changes",
	   "artist": "after life",
	   "description": "cool track",
	   "url": "https://oz-tf.nyc3.digitaloceanspaces.com/audio/unsaved-changes.mp3",
	   "filepath": "unsaved-changes.mp3"
	 }
   ]
 }`)

func TestCreateDB(t *testing.T) {
	db, err := ConnectDB(redisAddr)

	if err != nil {
		t.Fatalf("error connection to db: %e", err)
	}

	defer db.Conn.Close()

	res, err := db.Conn.Do("SET", "foo", "bar")

	if err != nil {
		t.Fatalf("error sending command to Redis: %e", err)
	}
	// spew.Dump(db.JsonHandler)

	t.Logf("Redis Connection Successful: %s", res)
}

func TestAddJsonStruct(t *testing.T) {
	db, err := ConnectDB(redisAddr)

	if err != nil {
		t.Fatalf("error connection to db: %e", err)
	}

	defer db.Conn.Close()

	plc := PlaylistContainer{}
	plc.PlaylistFromJson(jsonData)

	t.Logf("Playlist Name: %s", plc.Playlist.Name)

	res, err := db.AddJsonStruct("playlist", plc)

	if err != nil {
		t.Fatalf("Failure to add json struct %e", err)
	}

	t.Logf("Struct Added Successfully %s", res)

}

func TestGetJsonStruct(t *testing.T) {
	db, err := ConnectDB(redisAddr)

	jsonKey := "playlist"

	if err != nil {
		t.Fatalf("error connection to db: %e", err)
	}

	defer db.Conn.Close()

	plc := PlaylistContainer{}
	plc.PlaylistFromJson(jsonData)

	t.Logf("Playlist Name: %s", plc.Playlist.Name)

	res, err := db.AddJsonStruct(jsonKey, plc)

	if err != nil {
		t.Fatalf("Failure to SET json struct %e", err)
	}

	t.Logf("Struct Added Successfully %s", res)

	byteResp, err := db.GetJsonStruct(jsonKey)

	if err != nil {
		t.Fatalf("Failure to GET json struct %e", err)
	}

	t.Log("Json struct returned successfully")

	newPlc := PlaylistContainer{}
	// json.Unmarshal(byteResp, &newPlc)
	// FromJson unmarshals the entire playlist container
	newPlc.FromJson(byteResp)
	spew.Dump(newPlc)

	if newPlc.Playlist.Name == plc.Playlist.Name {
		t.Logf("Json struct successfully unmarshaled, %s", newPlc.Playlist.Name)
	} else {
		t.Errorf("Playlists do not match: %s != %s", newPlc.Playlist.Name, plc.Playlist.Name)
	}
}

func TestAppendAndGetSession(t *testing.T) {
	db, err := ConnectDB(redisAddr)
	if err != nil {
		t.Fatalf("error connection to db: %e", err)
	}
	uniqSession := uuid.New()
	_, err = db.AppendListSession(uniqSession.String())

	if err != nil {
		t.Fatalf("Failure to append session to list %e", err)
	}

	sessions, err := db.GetSessions()

	if err != nil {
		t.Fatalf("Failure to get sessions %e", err)
	}

	t.Logf("sessions: %v", sessions)

	if sessions[0] == uniqSession.String() {
		t.Logf("session %v pushed correctly", sessions[0])
	} else {
		t.Fatalf("incorrect session index in list: %v", sessions[0])
	}

	cs, err := db.GetCurrentSession()
	if cs == uniqSession.String() {
		t.Logf("current-session %v set correctly", cs)
	} else {
		t.Fatalf("current session not set: %v", cs)
	}
}
