// A simple example exposing fictional RPC latencies with different types of
// random distributions (uniform, normal, and exponential) as Prometheus
// metrics.
package main

import (
	//"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/xackery/eqemuconfig"
)

var (
	addr              = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	uniformDomain     = flag.Float64("uniform.domain", 0.0002, "The domain for the uniform distribution.")
	normDomain        = flag.Float64("normal.domain", 0.0002, "The domain for the normal distribution.")
	normMean          = flag.Float64("normal.mean", 0.00001, "The mean for the normal distribution.")
	oscillationPeriod = flag.Duration("oscillation-period", 10*time.Minute, "The duration of the rate oscillation period.")
)

var (
	// Create a summary to track players online.
	onlineCount = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "online_count_minutes",
			Help:       "Online Count, every 60 seconds.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"service"},
	)
)

func init() {
	// Register the with Prometheus's default registry.
	prometheus.MustRegister(onlineCount)
}

func main() {

	option := ""
	config, err := eqemuconfig.GetConfig()
	if err != nil {
		log.Println("Error while loading eqemu_config.xml to start:", err.Error())
		fmt.Println("press a key then enter to exit.")

		fmt.Scan(&option)
		os.Exit(1)
	}

	db, err := sqlx.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true", config.Database.Username, config.Database.Password, config.Database.Host, config.Database.Port, config.Database.Db))
	if err != nil {
		log.Println("Error while connecting to db:", err.Error())
		fmt.Println("press a key then enter to exit.")

		fmt.Scan(&option)
		os.Exit(1)
		return
	}
	defer db.Close()

	flag.Parse()

	//start := time.Now()

	// Periodically record some sample latencies for the three services.
	go func() {
		onlineQuery := "select count(last_login) from character_data where last_login >= UNIX_TIMESTAMP(now()-600) LIMIT 1"
		count := float64(0)
		for {
			rows, err := db.Queryx(onlineQuery)
			if err != nil {
				log.Println("Error onlineQuery:", err.Error)
				time.Sleep(60 * time.Second)
				continue
			}
			for rows.Next() {
				err = rows.Scan(&count)
				if err != nil {
					log.Println("Error onlineQuery:", err.Error)
					time.Sleep(60 * time.Second)
					continue
				}
			}
			onlineCount.WithLabelValues("normal").Observe(count)
			time.Sleep(60 * time.Second)
		}
	}()

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*addr, nil))
}
