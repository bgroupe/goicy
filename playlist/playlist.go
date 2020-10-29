package playlist

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"

	"github.com/bgroupe/goicy/config"
	"github.com/bgroupe/goicy/logger"
	"github.com/bgroupe/goicy/util"
	"github.com/davecgh/go-spew/spew"
)

var playlist []string
var idx int
var np string
var nowPlaying Track
var currentSession string

var plc PlaylistContainer

func First() string {
	db, err := ConnectDB(config.Cfg.RedisURL)
	if err != nil {
		logger.Log("error connecting to db", 0)
		panic(err)
	}

	defer db.Conn.Close()

	if plc.PlaylistLength() > 0 {
		sessionNPKey := fmt.Sprintf("%v-now-playing", plc.Sessions[0])
		res, err := db.AddJsonStruct(sessionNPKey, plc.Playlist.Tracks[0])

		if e := logStructUpdate(err, res); e != nil {
			return ""
		} else {
			return plc.Playlist.Tracks[0].FilePath
		}
	} else {
		return ""
	}
}

func Next(pc PlaylistControl) string {
	db, err := ConnectDB(config.Cfg.RedisURL)
	if err != nil {
		logger.Log("error connecting to db", 0)
		panic(err)
	}

	defer db.Conn.Close()

	if idx > plc.PlaylistLength()-1 {
		idx = 0
	}

	nowPlaying = plc.Playlist.Tracks[idx]

	//TODO: Reload
	if pc.Reload {
		LoadJSON()
	}

	for (nowPlaying == plc.Playlist.Tracks[idx]) && (plc.PlaylistLength() > 1) {
		if !config.Cfg.PlayRandom {
			spew.Dump("idx BEFORE iteration", idx)
			idx = idx + 1
			spew.Dump("idx AFTER iteration", idx)
			if idx > plc.PlaylistLength()-1 {
				spew.Dump("idx greater than length", idx)
				idx = 0
			}
		} else {
			idx = rand.Intn(plc.PlaylistLength())
			spew.Dump("idx RANDOMIZED", idx)
		}
	}

	// TODO: handle errors in main function
	// DONE: Add session path for now playing
	// find current track by getting current session
	// LRANGE sessions 0 0
	// GET  <session>-now-playing
	sessionNPKey := fmt.Sprintf("%v-now-playing", plc.Sessions[0])
	res, err := db.AddJsonStruct(sessionNPKey, plc.Playlist.Tracks[idx])

	if e := logStructUpdate(err, res); e != nil {
		return ""
	} else {
		return plc.Playlist.Tracks[idx].FilePath
	}
}

// Loads json playlist file. Creates a dir configured by `basepath` which defaults to `tmp`
func LoadJSON() error {
	//  DONE: Load and save playlist
	db, err := ConnectDB(config.Cfg.RedisURL)
	if err != nil {
		logger.Log("error connecting to db", 0)
		return err
	}

	defer db.Conn.Close()

	if ok := util.FileExists(config.Cfg.Playlist); !ok {
		return errors.New("Playlist file doesn't exist")
	}

	jsonFile, err := os.Open(config.Cfg.Playlist)

	if err != nil {
		fmt.Println("error opening json file")
		return err
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	plc.PlaylistFromJson(byteValue)

	fd := NewDownloader(plc.Playlist.DlCfg)

	plc.AppendFileSession(fd.Session, fd.SessionPath)

	for i, track := range plc.Playlist.Tracks {
		dlf, err := fd.Download(track)
		if err != nil {
			return err
		}

		plc.UpdateTrackFilePath(dlf, i)
	}

	// append session to the master list
	_, err = db.AppendListSession(fd.Session)
	if err != nil {
		logger.Logf("Failure to append session to master list %v", 0, err.Error())
	}

	// session-playlist
	sessionPlaylistKey := fmt.Sprintf("%v-playlist", fd.Session)
	res, err := db.AddJsonStruct(sessionPlaylistKey, plc.Playlist)

	if err != nil {
		logger.Logf("Failure to add json struct %v", 0, err.Error())
	} else {
		logger.Logf("Added session-playlist: %v", 1, res.(string))
	}

	return err
}

func logStructUpdate(e error, r interface{}) error {
	if e != nil {
		logger.Logf("Failure to add json struct %v", 0, e.Error())
	} else {
		logger.Logf("Added current-playlist: %v", 1, r.(string))
	}

	return e
}

func RegisterPlaylistControl(pc PlaylistControl) error {
	db, err := ConnectDB(config.Cfg.RedisURL)

	if err != nil {
		logger.Log("error connecting to db", 0)
		return err
	}

	defer db.Conn.Close()

	_, err = db.AddJsonStruct("playlist-control", pc)
	if err != nil {
		logger.Log("error adding playlist control to db", 0)
		return err
	}
	return err
}

// MAYBE: https://github.com/teris-io/shortid
// TODO: use previous session
