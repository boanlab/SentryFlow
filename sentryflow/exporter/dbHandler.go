// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	cfg "github.com/5GSEC/SentryFlow/config"
	"github.com/5GSEC/SentryFlow/types"

	_ "github.com/mattn/go-sqlite3"
)

// MDB global reference for Sqlite3 Handler
var MDB *MetricsDBHandler

// MetricsDBHandler Structure
type MetricsDBHandler struct {
	db     *sql.DB
	dbFile string
}

// init Function
func init() {
	MDB = NewMetricsDBHandler()
}

// NewMetricsDBHandler Function
func NewMetricsDBHandler() *MetricsDBHandler {
	ret := &MetricsDBHandler{
		dbFile: cfg.GlobalCfg.MetricsDBFileName,
	}
	return ret
}

// InitMetricsDBHandler Function
func (md *MetricsDBHandler) InitMetricsDBHandler() bool {
	log.Printf("[DB] Using DB File as %s", md.dbFile)
	targetDir := filepath.Dir(md.dbFile)
	_, err := os.Stat(targetDir)
	if err != nil {
		log.Printf("[DB] Unable to find target directory %s, creating one...", targetDir)
		err := os.Mkdir(targetDir, 0750)
		if err != nil {
			log.Printf("[Error] Unable to create directory for metrics DB %s: %v", targetDir, err)
			return false
		}
	}

	md.db, err = sql.Open("sqlite3", md.dbFile)
	if err != nil {
		log.Printf("[Error] Unable to open metrics DB: %v", err)
		return false
	}

	err = md.initDBTables()
	if err != nil {
		log.Printf("[Error] Unable to initialize metrics DB tables: %v", err)
		return false
	}

	return true
}

// StopMetricsDBHandler Function
func (md *MetricsDBHandler) StopMetricsDBHandler() {
	_ = md.db.Close()
}

// initDBTables Function
func (md *MetricsDBHandler) initDBTables() error {
	_, err := md.db.Exec(`
		CREATE TABLE IF NOT EXISTS access_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			labels BLOB,
			annotations BLOB
		);

		CREATE TABLE IF NOT EXISTS aggregated_access_logs (
			id INTEGER PRIMARY KEY,
			log_id INTEGER,
			log_data BLOB,
			FOREIGN KEY (log_id) REFERENCES access_log(id)
		);
	
		CREATE TABLE IF NOT EXISTS per_api_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api TEXT,
			count INTEGER
		);
	`)

	return err
}

// PerAPICountInsert Function
func (md *MetricsDBHandler) AccessLogInsert(data types.DbAccessLogType) error {
	var id int64
	//var logData [][]byte
	exist := md.db.QueryRow("SELECT id FROM access_log WHERE labels = ? AND annotations = ?", data.Labels, data.Annotations).Scan(&id)

	if exist != nil {
		result, err := md.db.Exec("INSERT INTO access_log (labels, annotations) VALUES (?, ?)", data.Labels, data.Annotations)
		if err != nil {
			log.Printf("INSERT accesslog error: %v", err)
			return err
		}

		lastId, err := result.LastInsertId()
		if err != nil {
			log.Printf("INSERT accesslog error: %v", err)
			return err
		}

		id = lastId
	}

	_, err := md.db.Exec("INSERT INTO aggregated_access_logs (log_id, log_data) VALUES (?, ?)", id, data.AccesLog)
	if err != nil {
		log.Printf("INSERT accesslog error: %v", err)
	}

	return err
}

func (md *MetricsDBHandler) AggregatedAccessLogSelect() (map[int64][][]byte, error) {
	als := make(map[int64][][]byte)

	rows, err := md.db.Query("SELECT id FROM access_log")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var accessLogID int64
		log.Printf("!!!!@#!@#!@#!@#!d;klfasdk;fasdkf %d", accessLogID)
		if err := rows.Scan(&accessLogID); err != nil {
			log.Fatal(err)
		}

		aggregatedRows, err := md.db.Query("SELECT log_data FROM aggregated_access_logs WHERE log_id = ?", accessLogID)
		if err != nil {
			log.Fatal(err)
		}
		defer aggregatedRows.Close()

		var logDataList [][]byte
		for aggregatedRows.Next() {
			var logData []byte
			if err := aggregatedRows.Scan(&logData); err != nil {
				log.Fatal(err)
			}
			log.Printf("wowowowow!!!!@#!@#!@#!@#!d;klfasdk;fasdkf %v", logData)
			logDataList = append(logDataList, logData)
		}
		als[accessLogID] = logDataList
	}
	log.Printf("adfjasdkfasd;klfasdk;fasdkf %d", len(als))
	return als, err
}

// PerAPICountInsert Function
func (md *MetricsDBHandler) PerAPICountInsert(data types.PerAPICount) error {
	var existAPI int
	err := md.db.QueryRow("SELECT COUNT(*) FROM per_api_metrics WHERE api = ?", data.Api).Scan(&existAPI)
	if err != nil {
		return err
	}

	if existAPI == 0 {
		_, err := md.db.Exec("INSERT INTO per_api_metrics (api, count) VALUES (?, ?)", data.Api, data.Count)
		if err != nil {
			return err
		}
	} else {
		err := md.PerAPICountUpdate(data)
		if err != nil {
			return err
		}
	}

	return err
}

// PerAPICountSelect Function
func (md *MetricsDBHandler) PerAPICountSelect(api string) (types.PerAPICount, error) {
	var tm types.PerAPICount

	err := md.db.QueryRow("SELECT api, count FROM per_api_metrics WHERE api = ?", api).Scan(&tm.Api, &tm.Count)
	if err != nil {
		return tm, err
	}

	return tm, err
}

// PerAPICountDelete Function
func (md *MetricsDBHandler) PerAPICountDelete(api string) error {
	_, err := md.db.Exec("DELETE FROM per_api_metrics WHERE api = ?", api)
	if err != nil {
		return err
	}

	return err
}

// PerAPICountUpdate Function
func (md *MetricsDBHandler) PerAPICountUpdate(data types.PerAPICount) error {
	var existAPI int
	err := md.db.QueryRow("SELECT COUNT(*) FROM per_api_metrics WHERE api = ?", data.Api).Scan(&existAPI)
	if err != nil {
		return err
	}

	if existAPI > 0 {
		_, err = md.db.Exec("UPDATE per_api_metrics SET count = ? WHERE api = ?", data.Count, data.Api)
		if err != nil {
			return err
		}
	}

	return nil
}
