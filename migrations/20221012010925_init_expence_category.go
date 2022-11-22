package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upInitExpenceCategory, downInitExpenceCategory)
}

func upInitExpenceCategory(tx *sql.Tx) error {
	const query = `
	CREATE TABLE expence_category 
	(
		id bigint PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
		user_id bigint REFERENCES users (id) ON DELETE CASCADE,
		name text,
		UNIQUE (user_id, name)
	);
	`

	_, err := tx.Exec(query)

	return err
}

func downInitExpenceCategory(tx *sql.Tx) error {
	const query = `
	DROP TABLE expence_category; 
	`
	_, err := tx.Exec(query)
	return err
}