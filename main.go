// A simple example exposing fictional RPC latencies with different types of
// random distributions (uniform, normal, and exponential) as Prometheus
// metrics.
package main

import (
	"database/sql"
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
	addr              = flag.String("listen-address", ":8081", "The address to listen on for HTTP requests.")
	uniformDomain     = flag.Float64("uniform.domain", 0.0002, "The domain for the uniform distribution.")
	normDomain        = flag.Float64("normal.domain", 0.0002, "The domain for the normal distribution.")
	normMean          = flag.Float64("normal.mean", 0.00001, "The mean for the normal distribution.")
	oscillationPeriod = flag.Duration("oscillation-period", 10*time.Minute, "The duration of the rate oscillation period.")
)

//build card trackers
type cardTracker struct {
	gauge *prometheus.GaugeVec
	name  string
	id    int
}

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

	currencyCount = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "currency_count_minutes",
			Help:       "Currency Count, every 60 seconds.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"service"},
	)

	radiantCount = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "radiant_count_minutes",
			Help:       "Radiant Crystal Count, every 60 seconds.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"service"},
	)

	ebonCount = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "ebon_count_minutes",
			Help:       "Ebon Crystal Count, every 60 seconds.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"service"},
	)

	expCount = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "exp_count_minutes",
			Help:       "Experience Count, every 60 seconds.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"service"},
	)

	cards []cardTracker
)

func init() {

	cardMap := map[string]int{
		"dragon":       100100,
		"insect":       100101,
		"animal":       100102,
		"construct":    100103,
		"extra_planar": 100104,
		"giant":        100105,
		"humanoid":     100106,
		"lycanthrope":  100107,
		"magical":      100108,
		"monster":      100109,
		"plant":        100110,
		"summoned":     100111,
		"undead":       100112,
		"gnoll":        100113,
		"aviak":        100114,
		"werewolf":     100115,
		"kobold":       100116,
		"orc":          100117,
		"fungus":       100118,
		"goblin":       100119,
		"evil_eye":     100120,
		"human":        100121,
		"barbarian":    100122,
		"erudite":      100123,
		"wood_elf":     100124,
		"high_elf":     100125,
		"dark_elf":     100126,
		"half_elf":     100127,
		"dwarf":        100128,
		"troll":        100129,
		"ogre":         100130,
		"halfling":     100131,
		"gnome":        100132,
		"froglok":      100133,
		"shadowed_man": 100134,
		"spider":       100135,
		"beetle":       100136,
		"snake":        100137,
		"wolf":         100138,
		"bear":         100139,
		"ghoul":        100140,
		"zombie":       100141,
		"skeleton":     100142,
		"chromadrac":   100143,
	}

	for name, value := range cardMap {
		card := cardTracker{
			id:   value,
			name: name,
			gauge: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: fmt.Sprintf("card_%s_count", name),
					Help: fmt.Sprintf("Total number of %s on server.", name),
				},
				[]string{"service"},
			),
		}
		cards = append(cards, card)
		prometheus.MustRegister(card.gauge)
	}

	// Register the with Prometheus's default registry.
	prometheus.MustRegister(onlineCount)
	prometheus.MustRegister(currencyCount)
	prometheus.MustRegister(radiantCount)
	prometheus.MustRegister(ebonCount)
	prometheus.MustRegister(expCount)
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

	// CARD
	go func() {
		var count sql.NullFloat64
		for {
			for _, card := range cards {
				err = db.QueryRow(`select SUM(inventory.charges) + sum(sharedbank.charges) 
					from inventory INNER JOIN sharedbank ON sharedbank.itemid = inventory.itemid 
					where inventory.itemid = ? or inventory.augslot1 = ?
					or sharedbank.augslot1 = ?`, card.id, card.id, card.id).Scan(&count)
				if err != nil {
					log.Println("Error exec card:", err)
					break
				}
				card.gauge.WithLabelValues("count").Set(count.Float64)
			}
			time.Sleep(60 * time.Second)
		}

	}()

	// ONLINE
	go func() {
		onlineQuery := "select count(last_login) from character_data where last_login >= UNIX_TIMESTAMP(now()-600) LIMIT 1"
		count := float64(0)
		for {
			rows, err := db.Queryx(onlineQuery)
			if err != nil {
				log.Println("Error onlineQuery:", err)
				time.Sleep(60 * time.Second)
				continue
			}
			for rows.Next() {
				err = rows.Scan(&count)
				if err != nil {
					log.Println("Error onlineQueryscan:", err)
					time.Sleep(60 * time.Second)
					continue
				}
			}
			onlineCount.WithLabelValues("normal").Observe(count)
			time.Sleep(60 * time.Second)
		}
	}()

	// EXPERIENCE
	go func() {
		expQuery := "select sum(exp)+sum(cc.exp_pool) from character_data cd INNER JOIN character_custom cc ON cc.character_id = cd.id;"
		expTotal := float64(0)
		for {
			rows, err := db.Queryx(expQuery)
			if err != nil {
				log.Println("Error expQuery:", err)
				time.Sleep(60 * time.Second)
				continue
			}
			for rows.Next() {
				err = rows.Scan(&expTotal)
				if err != nil {
					log.Println("Error expQueryScan:", err)
					time.Sleep(60 * time.Second)
					continue
				}
			}
			expCount.WithLabelValues("normal").Observe(expTotal)
			time.Sleep(60 * time.Second)
		}
	}()

	// CURRENCY
	go func() {
		currencyQuery := `SELECT a.sharedplat, cc.* FROM character_data cd 
INNER JOIN account a ON a.id = cd.account_id  
INNER JOIN character_currency cc ON cc.id = cd.id
WHERE a.status < 150 `
		type currencyRecord struct {
			Id                      int `db:"id"`
			Sharedplat              int `db:"sharedplat"`
			Platinum                int `db:"platinum"`
			Gold                    int `db:"gold"`
			Silver                  int `db:"silver"`
			Copper                  int `db:"copper"`
			Platinum_bank           int `db:"platinum_bank"`
			Gold_bank               int `db:"gold_bank"`
			Silver_bank             int `db:"silver_bank"`
			Copper_bank             int `db:"copper_bank"`
			Platinum_cursor         int `db:"platinum_cursor"`
			Gold_cursor             int `db:"gold_cursor"`
			Silver_cursor           int `db:"silver_cursor"`
			Copper_cursor           int `db:"copper_cursor"`
			Radiant_crystals        int `db:"radiant_crystals"`
			Ebon_crystals           int `db:"ebon_crystals"`
			Career_radiant_crystals int `db:"career_radiant_crystals"`
			Career_ebon_crystals    int `db:"career_ebon_crystals"`
		}
		goldMod := 10
		silverMod := 100
		copperMod := 1000

		for {
			totalCurrency := &currencyRecord{}
			rows, err := db.Queryx(currencyQuery)
			if err != nil {
				log.Println("Error currencyQuery:", err)
				time.Sleep(60 * time.Second)
				continue
			}

			for rows.Next() {

				currency := &currencyRecord{}
				err = rows.StructScan(&currency)
				if err != nil {
					log.Println("Error currencyQuery scan:", err)
					time.Sleep(60 * time.Second)
					continue
				}
				totalCurrency.Sharedplat += currency.Sharedplat
				totalCurrency.Gold += currency.Gold
				totalCurrency.Silver += currency.Silver
				totalCurrency.Copper += currency.Copper
				totalCurrency.Platinum_bank += currency.Platinum_bank
				totalCurrency.Gold_bank += currency.Gold_bank
				totalCurrency.Silver_bank += currency.Silver_bank
				totalCurrency.Copper_bank += currency.Copper_bank
				totalCurrency.Platinum_cursor += currency.Platinum_cursor
				totalCurrency.Gold_cursor += currency.Gold_cursor
				totalCurrency.Silver_cursor += currency.Silver_cursor
				totalCurrency.Copper_cursor += currency.Copper_cursor
				totalCurrency.Radiant_crystals += currency.Radiant_crystals
				totalCurrency.Ebon_crystals += currency.Ebon_crystals
			}
			totalPlatinum := float64(0)
			totalPlatinum += float64(totalCurrency.Sharedplat)
			totalPlatinum += float64(totalCurrency.Gold / goldMod)
			totalPlatinum += float64(totalCurrency.Silver / silverMod)
			totalPlatinum += float64(totalCurrency.Copper / copperMod)
			totalPlatinum += float64(totalCurrency.Platinum_bank)
			totalPlatinum += float64(totalCurrency.Gold_bank / goldMod)
			totalPlatinum += float64(totalCurrency.Silver_bank / silverMod)
			totalPlatinum += float64(totalCurrency.Copper_bank / copperMod)
			totalPlatinum += float64(totalCurrency.Platinum_cursor)
			totalPlatinum += float64(totalCurrency.Gold_cursor / goldMod)
			totalPlatinum += float64(totalCurrency.Silver_cursor / silverMod)
			totalPlatinum += float64(totalCurrency.Copper_cursor / copperMod)
			//log.Println(totalPlatinum)
			currencyCount.WithLabelValues("normal").Observe(float64(totalPlatinum))
			ebonCount.WithLabelValues("normal").Observe(float64(totalCurrency.Ebon_crystals))
			radiantCount.WithLabelValues("normal").Observe(float64(totalCurrency.Radiant_crystals))
			time.Sleep(60 * time.Second)
		}
	}()

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*addr, nil))
}
