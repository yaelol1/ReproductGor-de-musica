// package model mines mp3 files from a given folder and adds them to a database,
// also has the libraries to play the mp3 files.
package main

import (
	// "fmt"
	// "errors"
	"strconv"
	"github.com/bogem/id3v2/v2"
	"log"
	"path/filepath"
	"os"
	// -- testing home path
	// "os/user"

)

// A Song is a file that contains the basic info of the song.
type Song struct {
	id_song int
	performers string
	album string
	path string
	title string
	year int
	genre string
}

// An Album is a collection of songs
type Album struct {
	id_album int
	name string
	path string
	year int
}

// TODO: performer or a band and a artist
type Performer struct {
	id_performer int
	id_type int
	name string
}

// Addable checks that the preformer has the fields needed to be added to a database.
func (performer *Performer) Addable() bool{
	return performer.name != "" && (performer.id_type <= 4 && performer.id_type >= 0)
}

// addable checks that the album has every field except an id, so it can be added to the database.
func (album *Album) Addable() bool{
	return album.name != "" && album.path != ""
}

// mine walks recursively the path given, to find every .mp3 song, to store it
// in the database given.
func Mine(path string, database *Database) {
	// home, err := user.Current()
	// if err != nil {
	// 	log.Fatal("could not retrieve the current user:", err)
	// }
	// homePath := home.HomeDir
	// start := homePath + "/Music"

	// TODO: only traverses $HOME/Music
	log.Printf("DEBUG: Mine mining path: %v", path)
	filepath.Walk(path, func(path string, info os.FileInfo, _ error) error {
		log.Printf("Walking: %v, info: %v", path, info)
		if !info.IsDir() {
			log.Printf("Walking: %v", path)
			newSong, err := NewSong(path, info)

			// The file didn't have any tags so it wont be stored
			if err != nil {
				log.Print( err )
				return err
			}

			log.Printf("DEBUG: mine NewSong: %v", newSong)
			// TODO: Does it affect? if go
			 database.AddSong( newSong )
		}
		return nil
	})
}

// NewSong takes a file and its path to create a Song Struct and return it.
func NewSong(path string, info os.FileInfo) (*Song, error){

	// opening id3 tag
	tag, err := id3v2.Open( path , id3v2.Options{Parse: true} )
	if err != nil {
		return nil, err
	}

	defer tag.Close()

	year, _ := strconv.Atoi( tag.Year() )

	return &Song{
		performers: tag.Artist(),
		album: tag.Album(),
		path: path,
		title: tag.Title(),
		year: year,
		genre: tag.Genre(),
	}, nil
}
