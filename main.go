package main

import (
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var db *sqlx.DB
	var err error
	db, err = initializeDB()
	if err != nil {
		fmt.Println("Error initializing DB", err.Error())
	}
	schema := `CREATE TABLE test (
    test text,
    city text NULL,
    telcode integer);`
	result, err := db.Exec(schema)
	if err != nil {
		fmt.Println("Error creating table:", err.Error())
		return
	}
	rowCount, err := result.RowsAffected()
	if err != nil {
		fmt.Println("Error getting rows affected", err.Error())
	}
	if rowCount < 1 {
		fmt.Println("Last transaction failed, no rows affected")
	}
	fmt.Println("Done!")
}

func initializeDB() (db *sqlx.DB, err error) {
	var sdb *sql.DB
	sdb, err = sql.Open("sqlite3", "./test.db") //":memory:")
	if err != nil {
		return
	}
	db = sqlx.NewDb(sdb, "graph")

	err = db.Ping()
	if err != nil {
		return
	}
	return
}
