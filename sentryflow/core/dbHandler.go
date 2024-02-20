// SPDX-License-Identifier: Apache-2.0
package core

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type MetricsDB struct {
	db *sql.DB
}

type Metric struct {
	api   string
	count int
}

var metricsDB *MetricsDB

func (md *MetricsDB) insertMetric(api string, count int) error {
	var existAPI int
	err := md.db.QueryRow("SELECT COUNT(*) FROM metrics WHERE api = ?", api).Scan(&existAPI)
	if err != nil {
		panic(err)
	}

	if existAPI == 0 {
		_, err := md.db.Exec("INSERT INTO metrics (api, count) VALUES (?, ?)", api, count)
		if err != nil {
			panic(err)
		}
	} else {
		md.updateMetric(api, count)
	}
	return err
}

func (md *MetricsDB) seleteMetric(api string) (Metric, error) {
	var tempMetric Metric
	err := md.db.QueryRow("SELECT api, count FROM metrics WHERE api = ?", api).Scan(&tempMetric.api, &tempMetric.count)
	if err != nil {
		panic(err)
	}

	return tempMetric, err
}

func (md *MetricsDB) deleteMetric(api string) error {
	_, err := md.db.Exec("DELETE FROM metrics WHERE api = ?", api)
	if err != nil {
		panic(err)
	}

	return err
}

func (md *MetricsDB) updateMetric(api string, count int) error {
	var existAPI int
	err := md.db.QueryRow("SELECT COUNT(*) FROM metrics WHERE api = ?", api).Scan(&existAPI)
	if err != nil {
		panic(err)
	}

	if existAPI > 0 {
		_, err = md.db.Exec("UPDATE metrics SET count = ? WHERE api = ?", count, api)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Successfully updated age of %s to %d\n", api, count)
	} else {
		fmt.Printf("%s does not exist in the database\n", api)
	}

	return err
}

func InitMetricsDB() {
	db, err := sql.Open("sqlite3", "./example.db")
	if err != nil {
		panic(err)
	}

	metricsDB = &MetricsDB{db: db}

	_, err = metricsDB.db.Exec(`
		CREATE TABLE IF NOT EXISTS metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api TEXT,
			count INTEGER
		)
	`)
	if err != nil {
		panic(err)
	}
}
