package main

import (
	// "fmts "
	"strconv"
	"errors"
	"log"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	Database *sql.DB
	Path string
}

// CreateDatabase creates a database with the tables and rows needed.
func NewDatabase (path string) *Database{
	database, _ := sql.Open("sqlite3", path)

	// execute
	tables := ` CREATE TABLE types ( id_type INTEGER PRIMARY KEY , description TEXT );`
	create, _ := database.Prepare(tables)
	create.Exec()

	tables = ` INSERT INTO types VALUES (0 , "Person");`
	create, _ = database.Prepare(tables)
	create.Exec()


	tables = ` INSERT INTO types VALUES (1 , "Group"); `
	create, _ = database.Prepare(tables)
	create.Exec()

	tables = ` INSERT INTO types VALUES (2 , "Unknown");`
	create, _ = database.Prepare(tables)
	create.Exec()

	tables = ` CREATE TABLE performers ( id_performer INTEGER PRIMARY KEY , id_type INTEGER ,
		    name TEXT , FOREIGN KEY ( id_type ) REFERENCES types ( id_type ) );`
	create, _ = database.Prepare(tables)
	create.Exec()

	tables = ` CREATE TABLE persons ( id_person INTEGER PRIMARY KEY , stage_name TEXT , real_name TEXT ,
		     birth_date TEXT , death_date TEXT );`
	create, _ = database.Prepare(tables)
	create.Exec()

	tables = ` CREATE TABLE groups ( id_group INTEGER PRIMARY KEY , name TEXT , start_date TEXT , end_date TEXT );`
	create, _ = database.Prepare(tables)
	create.Exec()

	tables = ` CREATE TABLE albums ( id_album INTEGER PRIMARY KEY , path TEXT , name TEXT , year INTEGER );`
	create, _ = database.Prepare(tables)
	create.Exec()

	tables = ` CREATE TABLE rolas ( id_rola INTEGER PRIMARY KEY , id_performer INTEGER , id_album INTEGER ,
		    path TEXT , title TEXT , track INTEGER , year INTEGER , genre TEXT , FOREIGN KEY ( id_performer )
		    REFERENCES performers ( id_performer ) , FOREIGN KEY ( id_album ) REFERENCES albums ( id_album ) );`
	create, _ = database.Prepare(tables)
	create.Exec()

	tables = ` CREATE TABLE in_group ( id_person INTEGER , id_group INTEGER , PRIMARY KEY ( id_person , id_group ) ,
		    FOREIGN KEY ( id_person ) REFERENCES persons ( id_person ) , FOREIGN KEY ( id_group ) REFERENCES groups ( id_group ) ); `

	create, _ = database.Prepare(tables)
	create.Exec()

	return &Database{
		Database: database,
		Path: path,
	}
}

// OpenDatabase opens the sqlite3 database with the given path to return a Database struct.
func OpenDatabase (path string) (*Database, error){
	database, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	return &Database{
		Database: database,
		Path: path,
	}, nil
}

// AddSong adds a song to the database, returns true if the song was
// added or false if it was already in the database.
func (database *Database) AddSong(songToAdd *Song) bool{
	log.Printf("DEBUG: addsong findsong ENTER")
	if _, err := database.FindSong(songToAdd); err == nil{
		return false
	}
	log.Printf("DEBUG: addsong findsong EXIT")

	statement, _ := database.Database.Prepare(`INSERT INTO rolas (id_performer, id_album, path, title, track, year, genre)
	VALUES (?, ?, ?, ?, ?, ?, ?)`)

	// Adding or finding the album
	album := &Album{
		name: songToAdd.album,
		path: songToAdd.path,
		year: songToAdd.year,
	}

	log.Printf("DEBUG: addSong addAlbum ENTER")
	album, _ = database.addAlbum(album)
	id_album := album.id_album
	log.Printf("DEBUG: addsong addAlbum EXIT, final album: %v", album)

	// Adding or finding the performer
	performer := &Performer{
		name: songToAdd.performers,
		id_type: 3,
	}

	log.Printf("DEBUG: addSong addPerformer ENTER")
	performer, _ = database.addPerformer(performer)
	id_performer := performer.id_performer
	log.Printf("DEBUG: addSong addPerformer EXIT, final performer:", performer)

	statement.Exec(id_performer, id_album, songToAdd.path, songToAdd.title, 0, songToAdd.year, songToAdd.genre)
	log.Printf("DEBUG: addSong statement EXEC")
	return true
}

// FindSong finds a Song in the database and returns the song.
func (database *Database) FindSong(songToSearch *Song) (*Song, error){
	// The album given has an id.
	if songToSearch.id_song != 0  {
		_, err := database.findSongById(songToSearch)
		if err == nil {
			return songToSearch, nil
		}
	}
	log.Printf("DEBUG: findSong if -> else")

	// The Album doesn't have an id, nor a name, therefore cannot be searched.
	if songToSearch.title == "" {
		return nil, errors.New("The song must have a Title or an id to be found in the database.")
	}

	// The album will be searched by title
	var id_song, id_performer, id_album, year int
	var path, title, genre string

	// prepare statement begin
	stmtStr := "SELECT id_rola, id_performer, id_album, path, title, year, genre FROM rolas WHERE path = ? AND title = ?"
	tx, stmt := database.PrepareStatement(stmtStr)
	defer stmt.Close()

	// Query
	rows, err := stmt.Query(songToSearch.path, songToSearch.title)
	if err != nil {
		log.Fatal("could not execute query: ", err)
	}
	defer rows.Close()

	// Scan the rows
	for rows.Next() {
		err := rows.Scan(&id_song, &id_performer, &id_album, &path, &title, &year, &genre)
		log.Printf("id: %v, %v, %v, path: %v, title: %v, year: %v, genre: %v", id_song, id_performer, id_album, path, title, year, genre )
		if err != nil {
			log.Printf("DEBUG: findAlbumById album queryRow nil")
			log.Print(err)
			return nil, err
		}
	}

	log.Printf("DEBUG: findSong song found: " )
	// Check for errors and commit
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()

	performer := &Performer{
		name: songToSearch.performers,
		id_performer: id_performer,
	}

	album := &Album{
		name: songToSearch.album,
		path: songToSearch.path,
		year: songToSearch.year,
	}

	album, _ = database.addAlbum(album)

	performer, _ = database.addPerformer(performer)

	// Store results
	songToSearch.id_song = id_song
	songToSearch.performers = performer.name
	// songToSearch.id_album = id_album
	songToSearch.album = album.name
	songToSearch.path = path
	songToSearch.title = title
	songToSearch.year = year
	songToSearch.genre = genre


	log.Printf("DEBUG: findSong song found: %v", songToSearch )
	return songToSearch, nil
}

// addPerformer adds a performer if it doesn't exist.
func (database *Database) addPerformer(performer *Performer) (*Performer, error) {
	if performerFound, err := database.findPerformer(performer); err == nil{
		return performerFound, nil
	}

	if !performer.Addable() {
		return nil, errors.New("The performer does not have enough data to be added")
	}

	statement, _ := database.Database.Prepare(`INSERT INTO performers (id_type, name)
	VALUES (?, ?)`)
	statement.Exec(performer.id_type, performer.name)

	performer, _ = database.findPerformer(performer)

	return performer, nil
}

func (database *Database) findSongById(songToSearch *Song) (*Song, error){
	return nil, nil
}

// findPerformer adds a performer if it doesn't exist.
func (database *Database) findPerformer(performer *Performer) (*Performer, error) {
	if performer.id_performer != 0  {
		if performerFound, err := database.findPerformerById(performer); err == nil{
			return performerFound, nil
		}
	}

	// The Performer doesn't have an id, nor a name, therefore cannot be searched.
	if performer.name == "" {
		return nil, errors.New("The Performer must have a Name or an id to be found in the database.")
	}

	// The performer will be searched by name
	var id_performer, id_type int

	// prepare statement begin
	stmtStr := "SELECT id_performer, id_type FROM performers WHERE name = ?"
	tx, stmt := database.PrepareStatement(stmtStr)
	defer stmt.Close()

	// Query
	rows, err := stmt.Query(performer.name)
	if err != nil {
		log.Fatal("could not execute query: ", err)
	}
	defer rows.Close()

	// Scan the rows
	for rows.Next() {
		err := rows.Scan(&id_performer, &id_type)
		if err != nil {
			log.Printf("DEBUG: findAlbumById album queryRow nil")
			log.Print(err)
			return nil, err
		}
	}

	// Check for errors and commit
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()

	// Store values
	performer.id_type = id_type
	performer.id_performer = id_performer

	return performer, nil
}

func (database *Database) findPerformerById(performer *Performer) (*Performer, error) {
	// The performer will be searched by name
	var id_type int
	var name string

	// prepare statement begin
	stmtStr := "SELECT name, id_type FROM performers WHERE id_performer = ? "
	tx, stmt := database.PrepareStatement(stmtStr)
	defer stmt.Close()

	// Query
	rows, err := stmt.Query(performer.id_performer)
	if err != nil {
		log.Fatal("could not execute query: ", err)
	}
	defer rows.Close()

	// Scan the rows
	for rows.Next() {
		err := rows.Scan(&name, &id_type)
		if err != nil {
			log.Printf("DEBUG: findAlbumById album queryRow nil")
			log.Print(err)
			return nil, err
		}
	}

	// Check for errors and commit
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()

	// Store values
	performer.name = name
	performer.id_type = id_type

	return performer, nil
}

// addAlbum adds an album if it doesnt' exist
func (database *Database) addAlbum(album *Album) (*Album, error) {

	log.Printf("DEBUG: addAlbum album: %v", album)
	log.Printf("DEBUG: addAlbum findAlbum enter")
	if albumFound, err := database.findAlbum(album); err == nil{
		log.Printf("DEBUG: addAlbum findAlbum inside return error album found: %v", albumFound)
		return albumFound, errors.New("Album already in database")
	}
	log.Printf("DEBUG: findAlbum exit")

	if !album.Addable() {
		return nil, errors.New("Album needs a name, path and year to be added")
	}


	log.Printf("DEBUG: addAlbum insert ")
	statement, err := database.Database.Prepare(`INSERT INTO albums (name, path, year)
	VALUES (?, ?, ?)`)
	if err != nil {
		log.Print(err)
	}
	statement.Exec(album.name, album.path, album.year)
	log.Printf("DEBUG: addAlbum insert exit ")

	_, findErr := database.findAlbum(album)
	if findErr != nil {
		log.Print("Fatal Error:")
		log.Fatal(err)
	}
	log.Printf("DEBUG: addAlbum album: %v", album)

	return album, nil
}

// findAlbum adds an album to the database and returns the album given with the database entries Album and an error.
// it the album given has an id, the album will be searched by id 
func (database *Database) findAlbum(album *Album) (*Album, error) {
	// The album given has an id.
	if album.id_album != 0 && album.year != 0 {
		_, err := database.findAlbumById(album)
		if err == nil {
			return album, nil
		}
	}
	log.Printf("DEBUG: findAlbum if -> else")

	// The Album doesn't have an id, nor a name, therefore cannot be searched.
	if album.name == "" {
		return nil, errors.New("The album must have a Name or an id to be found in the database.")
	}

	// The album will be searched by name
	var  id_album, year int
	var path string

	// prepare statement begin
	stmtStr := "SELECT id_album, path, year FROM albums WHERE name = ? LIMIT 1"
	tx, stmt := database.PrepareStatement(stmtStr)
	defer stmt.Close()

	// Query
	rows, err := stmt.Query(album.name)
	if err != nil {
		log.Fatal("could not execute query: ", err)
	}
	defer rows.Close()

	// Scan the rows
	for rows.Next() {
		err := rows.Scan(&id_album, &path, &year)
		if err != nil {
			log.Printf("DEBUG: findAlbumById album queryRow nil")
			log.Print(err)
			return nil, err
		}
	}

	// Check for errors and commit
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()


	// Store results
	log.Printf("DEBUG: findAlbum album = { %v , %v , %v }",id_album, path, year)

	album.id_album = id_album
	album.path = path
	album.year = year

	return album, nil
}

// findAlbumById searches for the album by id
func (database *Database) findAlbumById(album *Album) (*Album, error) {
	log.Printf("DEBUG: findAlbum if -> true")

	id := strconv.Itoa(album.id_album)
	stmtStr := "SELECT  path, name, year FROM albums WHERE id_album = ? LIMIT 1"

	// prepare statement
	tx, stmt := database.PrepareStatement(stmtStr)
	defer stmt.Close()

	// Query
	rows, err := stmt.Query(id)
	if err != nil {
		log.Fatal("could not execute query: ", err)
	}
	defer rows.Close()

	// result variables
	var  year int
	var path, name string

	// Scan the rows
	for rows.Next() {
		err := rows.Scan(&path, &name, &year)
		if err != nil {
			log.Printf("DEBUG: findAlbumById album queryRow nil")
			log.Print(err)
			return nil, err
		}
	}

	// Check for errors and commit
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()

	// Ensure the names are the same
	if album.name != name {
		return nil, errors.New("sql: Names do not match with id given")
	}
	// Store the results and commit
	log.Printf("DEBUG: findAlbum album = { %v , %v , %v  }", name, path, year)

	album.path = path
	album.name = name
	album.year = year

	return album, nil

}


// PrepareStatement initializes an sqlite prepared statement from a string
// and returns the corresponding sql context and prepared statement.
func (database *Database) PrepareStatement(statement string) (*sql.Tx, *sql.Stmt) {
	tx, err := database.Database.Begin()
	if err != nil {
		log.Fatal("could not begin transaction: ", err)
	}
	stmt, err := tx.Prepare(statement)
	if err != nil {
		log.Fatal("could not prepare statement: ", err)
	}
	return tx, stmt
}
