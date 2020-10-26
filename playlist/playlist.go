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
)

const (
	playlistKey    = "current-playlist"
	nowPlayingKey  = "now-playing"
	tracksKey      = "tracks"
	currentSession = "current-session"
)

var playlist []string
var idx int
var np string
var nowPlaying Track

var plc PlaylistContainer

func First() string {
	db, err := ConnectDB(config.Cfg.RedisURL)
	if err != nil {
		logger.Log("error connecting to db", 0)
		panic(err)
	}

	defer db.Conn.Close()

	if plc.PlaylistLength() > 0 {
		res, err := db.AddJsonStruct("now-playing", plc.Playlist.Tracks[0])
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

	if pc.Reload {
		LoadJSON()
	}

	for (nowPlaying == plc.Playlist.Tracks[idx]) && (plc.PlaylistLength() > 1) {
		if !config.Cfg.PlayRandom {
			idx = idx + 1
			if idx > plc.PlaylistLength()-1 {
				idx = 0
			} else {
				idx = rand.Intn(plc.PlaylistLength())
			}
		}
	}
	// TODO: handle errors in main function
	// TODO: Add session path for now playing
	res, err := db.AddJsonStruct(nowPlayingKey, plc.Playlist.Tracks[0])
	if e := logStructUpdate(err, res); e != nil {
		return ""
	} else {
		return plc.Playlist.Tracks[idx].FilePath
	}
}

// Loads json playlist file. Creates a dir configured by `basepath` which defaults to `tmp`
func LoadJSON() error {
	//  TODO: Load and save playlist
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

	plc.AppendFileSession(fd.SessionPath)

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

// MAYBE: https://github.com/teris-io/shortid
