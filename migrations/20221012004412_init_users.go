package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upInitUsers, downInitUsers)
}

func upInitUsers(tx *sql.Tx) error {
	const query = `
	CREATE TABLE users 
	(
		id bigint PRIMARY KEY,
		base_currency_id smallint DEFAULT 0 REFERENCES currency (id),
		default_month_limit bigint DEFAULT 1000000,
		current_month_limit bigint DEFAULT 1000000
	);
	`

	_, err := tx.Exec(query)

	return err
}

func downInitUsers(tx *sql.Tx) error {
	const query = `
	DROP TABLE users; 
	`
	_, err := tx.Exec(query)
	return err
}
