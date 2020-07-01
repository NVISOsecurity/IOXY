package main

import (
	"database/sql"
	"os"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type SessionMgmt struct {
	client_id  string
	session_id string
	username   string
	password   string
}

type Messages struct {
	session_id string
	topic      string
	payload    string
	dup        bool
	qos        int
	retain     bool
	timestamp  string
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func createDBFile() {
	if _, err := os.Stat("database/mqtt.db"); os.IsNotExist(err) {
		os.MkdirAll("database", 0700)
		os.Create("database/mqtt.db")
	}
}

func initDB() (db *sql.DB) {

	createDBFile()
	db, _ = sql.Open("sqlite3", "database/mqtt.db")

	sessionTable := `
	CREATE TABLE IF NOT EXISTS session_mgmt(
		client_id VARCHAR(25) PRIMARY KEY,
		session_id VARCHAR(36),
		username TEXT,
		password TEXT
	);
	`
	messagetable := `CREATE TABLE IF NOT EXISTS messages(
		session_id VARCHAR(36),
		topic TEXT,
		payload TEXT,
		dup_flag BOOLEAN,
		qos_flag TINYINT,
		retain_flag BOOLEAN,
		timestamp TIMESTAMP
	);`
	stmt, err := db.Prepare(sessionTable)
	checkError(err)
	res, err := stmt.Exec()
	checkError(err)
	affectedRowsSession, err := res.RowsAffected()
	checkError(err)
	stmt, err = db.Prepare(messagetable)
	checkError(err)
	res, err = stmt.Exec()
	checkError(err)
	affectedRowsMessages, err := res.RowsAffected()
	if affectedRowsMessages > 0 || affectedRowsSession > 0 {
		log.Println("Database created")
	}
	return db
}

func addClient(db *sql.DB, client SessionMgmt) {
	query := `
	INSERT OR REPLACE INTO session_mgmt(
		client_id,
		session_id,
		username,
		password
	) values(?, ?, ?, ?)
	`
	stmt, err := db.Prepare(query)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	res, err := stmt.Exec(client.client_id, client.session_id, client.username, client.password)
	checkError(err)
	_, err = res.RowsAffected()
	checkError(err)
	log.Println("New client added to the database")
}

func addMessage(db *sql.DB, message Messages) {
	query := `
	INSERT INTO messages(
		session_id,
		topic,
		payload,
		dup_flag,
		qos_flag,
		retain_flag,
		timestamp
	) values(?, ?, ?, ?, ?, ?, ?)
	`
	stmt, err := db.Prepare(query)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	res, err := stmt.Exec(message.session_id, message.topic, message.payload, message.dup, message.qos, message.retain, message.timestamp)
	checkError(err)
	_, err = res.RowsAffected()
	checkError(err)
}
