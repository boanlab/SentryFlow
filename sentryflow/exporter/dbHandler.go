// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"time"

	cfg "github.com/5GSEC/SentryFlow/config"
	"github.com/5GSEC/SentryFlow/protobuf"
	"github.com/5GSEC/SentryFlow/types"
	"google.golang.org/protobuf/proto"

	"github.com/mattn/go-sqlite3"
)

// MDB global reference for Sqlite3 Handler
var MDB *MetricsDBHandler

// MetricsDBHandler Structure
type MetricsDBHandler struct {
	db          *sql.DB
	dbFile      string
	dbClearTime int
}

// AggregationData Structure
type AggregationData struct {
	Labels     string
	Namespace  string
	AccessLogs []string
}

// init Function
func init() {
	MDB = NewMetricsDBHandler()
}

// NewMetricsDBHandler Function
func NewMetricsDBHandler() *MetricsDBHandler {
	ret := &MetricsDBHandler{
		dbFile:      cfg.GlobalCfg.MetricsDBFileName,
		dbClearTime: cfg.GlobalCfg.MetricsDBClearTime,
	}
	return ret
}

// InitMetricsDBHandler Function
func (md *MetricsDBHandler) InitMetricsDBHandler() bool {
	libVersion, libVersionNumber, sourceID := sqlite3.Version()
	log.Printf("[DB] Using Sqlite Version is %v %v %v", libVersion, libVersionNumber, sourceID)
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

	go aggregationTimeTickerRoutine()
	go exportTimeTickerRoutine()
	go DBClearRoutine()

	return true
}

// StopMetricsDBHandler Function
func (md *MetricsDBHandler) StopMetricsDBHandler() {
	_ = md.db.Close()
}

// initDBTables Function
func (md *MetricsDBHandler) initDBTables() error {
	_, err := md.db.Exec(`
		CREATE TABLE IF NOT EXISTS aggregation_table (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			labels TEXT,
			namespace TEXT,
			accesslog BLOB
		);
	
		CREATE TABLE IF NOT EXISTS per_api_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api TEXT,
			count INTEGER
		);
	`)

	return err
}

// AccessLogInsert Function
func (md *MetricsDBHandler) AccessLogInsert(data types.DbAccessLogType) error {
	alData, err := proto.Marshal(data.AccessLog)
	if err != nil {
		return err
	}

	_, err = md.db.Exec("INSERT INTO aggregation_table (labels, namespace, accesslog) VALUES (?, ?, ?)", data.Labels, data.Namespace, alData)
	if err != nil {
		log.Printf("INSERT accesslog error: %v", err)
		return err
	}

	return err
}

// GetLabelNamespacePairs Function
func (md *MetricsDBHandler) GetLabelNamespacePairs() ([]AggregationData, error) {
	query := `
		SELECT labels, namespace
		FROM aggregation_table
		GROUP BY labels, namespace
	`

	rows, err := md.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pairs []AggregationData
	for rows.Next() {
		var labels, namespace string
		err := rows.Scan(&labels, &namespace)
		if err != nil {
			return nil, err
		}
		pair := AggregationData{
			Labels:    labels,
			Namespace: namespace,
		}

		pairs = append(pairs, pair)
	}
	return pairs, nil
}

// AggregatedAccessLogSelect Function
func (md *MetricsDBHandler) AggregatedAccessLogSelect() (map[string][]*protobuf.APILog, error) {
	als := make(map[string][]*protobuf.APILog)
	pairs, err := md.GetLabelNamespacePairs()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT accesslog
		FROM aggregation_table
		WHERE labels = ? AND namespace = ?
	`
	for _, pair := range pairs {
		curKey := pair.Labels + pair.Namespace
		rows, err := md.db.Query(query, pair.Labels, pair.Namespace)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var accessLogs []*protobuf.APILog
		for rows.Next() {
			var accessLog []byte
			err := rows.Scan(&accessLog)
			if err != nil {
				return nil, err
			}

			al := &protobuf.APILog{}
			err = proto.Unmarshal(accessLog, al)

			accessLogs = append(accessLogs, al)
		}
		als[curKey] = accessLogs
	}

	return als, err
}

// PerAPICountInsert Function
func (md *MetricsDBHandler) PerAPICountInsert(data *types.PerAPICount) error {
	var existAPI int
	err := md.db.QueryRow("SELECT COUNT(*) FROM per_api_metrics WHERE api = ?", data.API).Scan(&existAPI)
	if err != nil {
		return err
	}

	if existAPI == 0 {
		_, err := md.db.Exec("INSERT INTO per_api_metrics (api, count) VALUES (?, ?)", data.API, data.Count)
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

	err := md.db.QueryRow("SELECT api, count FROM per_api_metrics WHERE api = ?", api).Scan(&tm.API, &tm.Count)
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

	return nil
}

// PerAPICountUpdate Function
func (md *MetricsDBHandler) PerAPICountUpdate(data *types.PerAPICount) error {
	var existAPI int
	err := md.db.QueryRow("SELECT COUNT(*) FROM per_api_metrics WHERE api = ?", data.API).Scan(&existAPI)
	if err != nil {
		return err
	}

	if existAPI > 0 {
		_, err = md.db.Exec("UPDATE per_api_metrics SET count = ? WHERE api = ?", data.Count, data.API)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetAllMetrics Function
func (md *MetricsDBHandler) GetAllMetrics() (map[string]uint64, error) {
	metrics := make(map[string]uint64)

	rows, err := md.db.Query("SELECT api, count FROM per_api_metrics")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var metric types.PerAPICount
		err := rows.Scan(&metric.API, &metric.Count)
		if err != nil {
			return nil, err
		}
		metrics[metric.API] = metric.Count
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return metrics, nil
}

// ClearAllTable Function
func (md *MetricsDBHandler) ClearAllTable() error {
	_, err := md.db.Exec("DELETE FROM aggregation_table")
	if err != nil {
		log.Fatal(err)
		return err
	}
	log.Println("Data in 'aggregation_table' deleted successfully.")

	_, err = md.db.Exec("DELETE FROM per_api_metrics")
	if err != nil {
		log.Fatal(err)
		return err
	}
	log.Println("Data in 'per_api_metrics' deleted successfully.")

	return nil
}

// DBClearRoutine Function
func DBClearRoutine() error {
	ticker := time.NewTicker(time.Duration(MDB.dbClearTime) * time.Second)

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := MDB.ClearAllTable()
			if err != nil {
				log.Printf("[Error] Unable to Clear DB tables: %v", err)
				return err
			}
		}
	}

	return nil
}
