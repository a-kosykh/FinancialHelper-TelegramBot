package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upInitCurrency, downInitCurrency)
}

func upInitCurrency(tx *sql.Tx) error {
	const query = `
	CREATE TABLE currency 
	(
		id smallint PRIMARY KEY,
		code character(3) NOT NULL,
		rate real NOT NULL
	);
	`

	_, err := tx.Exec(query)

	return err
}

func downInitCurrency(tx *sql.Tx) error {
	const query = `
	drop table currency; 
	`
	_, err := tx.Exec(query)
	return err
}
