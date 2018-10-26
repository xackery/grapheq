package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/xackery/eqemuconfig"
	"gopkg.in/yaml.v2"
)

var (
	addr    = flag.String("listen-address", ":8081", "The address to listen on for HTTP requests.")
	iconfig *Config
)

// Config is the configuration of the grapheq
type Config struct {
	RetryDelay int
	Influx     *Influx
}

// Influx represents an influxdb endpoint
type Influx struct {
	URL      string
	Database string
	User     string
	Password string
}

//build card trackers
type cardTracker struct {
	name  string
	id    int
	count int
}

var (
	cards   []cardTracker
	cardMap = map[string]int{
		"dragon":      100100,
		"insect":      100101,
		"animal":      100102,
		"construct":   100103,
		"extraPlanar": 100104,
		"giant":       100105,
		"humanoid":    100106,
		"lycanthrope": 100107,
		"magical":     100108,
		"monster":     100109,
		"plant":       100110,
		"summoned":    100111,
		"undead":      100112,
		"gnoll":       100113,
		"aviak":       100114,
		"werewolf":    100115,
		"kobold":      100116,
		"orc":         100117,
		"fungus":      100118,
		"goblin":      100119,
		"evilEye":     100120,
		"human":       100121,
		"barbarian":   100122,
		"erudite":     100123,
		"woodElf":     100124,
		"highElf":     100125,
		"darkElf":     100126,
		"halfElf":     100127,
		"dwarf":       100128,
		"troll":       100129,
		"ogre":        100130,
		"halfling":    100131,
		"gnome":       100132,
		"froglok":     100133,
		"shadowedMan": 100134,
		"spider":      100135,
		"beetle":      100136,
		"snake":       100137,
		"wolf":        100138,
		"bear":        100139,
		"ghoul":       100140,
		"zombie":      100141,
		"skeleton":    100142,
		"chromadrac":  100143,
	}
)

func main() {
	iconfig = &Config{}
	f, err := os.Open("config.yml")
	d := yaml.NewDecoder(f)
	err = d.Decode(iconfig)
	if err != nil {
		log.Println("failed to open influx config", err.Error())
		os.Exit(1)
	}

	option := ""
	config, err := eqemuconfig.GetConfig()
	if err != nil {
		log.Println("Error while loading eqemuConfig.xml to start:", err.Error())
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
		var count int64
		for {
			metrics := make(map[string]interface{})

			for name, card := range cardMap {
				err = db.QueryRow(`select count(itemid) from inventory 
					where inventory.itemid = ? or inventory.augslot1 = ?`, card, card).Scan(&count)
				if err != nil {
					log.Println("Error exec card:", err)
					break
				}
				metrics[name] = count
			}
			err = sendMetrics("card", "card", metrics)
			if err != nil {
				fmt.Println("failed to send card metrics", err.Error())
			}
			time.Sleep(60 * time.Second)
		}

	}()

	// ONLINE onlineCountMinutes
	go func() {
		onlineQuery := "select count(last_login) from character_data where last_login >= UNIX_TIMESTAMP(now()-600) LIMIT 1"
		count := int64(0)
		metrics := make(map[string]interface{})
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
			metrics["online"] = count
			err = sendMetrics("online", "count", metrics)
			if err != nil {
				fmt.Println("failed to send card metrics", err.Error())
			}
			time.Sleep(60 * time.Second)
		}
	}()

	// EXPERIENCE //expCountMinutes
	go func() {
		expQuery := "select sum(exp)+sum(cc.exp_pool) from character_data cd INNER JOIN character_custom cc ON cc.character_id = cd.id;"
		count := int64(0)
		metrics := make(map[string]interface{})
		for {
			rows, err := db.Queryx(expQuery)
			if err != nil {
				log.Println("Error expQuery:", err)
				time.Sleep(60 * time.Second)
				continue
			}
			for rows.Next() {
				err = rows.Scan(&count)
				if err != nil {
					log.Println("Error expQueryScan:", err)
					time.Sleep(60 * time.Second)
					continue
				}
			}
			metrics["exp_total"] = count
			err = sendMetrics("online", "count", metrics)
			if err != nil {
				fmt.Println("failed to send card metrics", err.Error())
			}
			//expCount.WithLabelValues("normal").Observe(expTotal)
			time.Sleep(60 * time.Second)
		}
	}()

	// CURRENCY //currencyCountMinutes radiantCountMinutes ebonCountMinutes
	go func() {
		currencyQuery := `SELECT a.sharedplat, cc.* FROM character_data cd 
INNER JOIN account a ON a.id = cd.account_id  
INNER JOIN character_currency cc ON cc.id = cd.id
WHERE a.status < 150 `
		type currencyRecord struct {
			ID                    int `db:"id"`
			Sharedplat            int `db:"sharedplat"`
			Platinum              int `db:"platinum"`
			Gold                  int `db:"gold"`
			Silver                int `db:"silver"`
			Copper                int `db:"copper"`
			PlatinumBank          int `db:"platinum_bank"`
			GoldBank              int `db:"gold_bank"`
			SilverBank            int `db:"silver_bank"`
			CopperBank            int `db:"copper_bank"`
			PlatinumCursor        int `db:"platinum_cursor"`
			GoldCursor            int `db:"gold_cursor"`
			SilverCursor          int `db:"silver_cursor"`
			CopperCursor          int `db:"copper_cursor"`
			RadiantCrystals       int `db:"radiant_crystals"`
			EbonCrystals          int `db:"ebon_crystals"`
			CareerRadiantCrystals int `db:"career_radiant_crystals"`
			CareerEbonCrystals    int `db:"career_ebon_crystals"`
		}
		goldMod := 10
		silverMod := 100
		copperMod := 1000
		metrics := make(map[string]interface{})
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
				totalCurrency.PlatinumBank += currency.PlatinumBank
				totalCurrency.GoldBank += currency.GoldBank
				totalCurrency.SilverBank += currency.SilverBank
				totalCurrency.CopperBank += currency.CopperBank
				totalCurrency.PlatinumCursor += currency.PlatinumCursor
				totalCurrency.GoldCursor += currency.GoldCursor
				totalCurrency.SilverCursor += currency.SilverCursor
				totalCurrency.CopperCursor += currency.CopperCursor
				totalCurrency.RadiantCrystals += currency.RadiantCrystals
				totalCurrency.EbonCrystals += currency.EbonCrystals
			}
			totalPlatinum := float64(0)
			totalPlatinum += float64(totalCurrency.Sharedplat)
			totalPlatinum += float64(totalCurrency.Gold / goldMod)
			totalPlatinum += float64(totalCurrency.Silver / silverMod)
			totalPlatinum += float64(totalCurrency.Copper / copperMod)
			totalPlatinum += float64(totalCurrency.PlatinumBank)
			totalPlatinum += float64(totalCurrency.GoldBank / goldMod)
			totalPlatinum += float64(totalCurrency.SilverBank / silverMod)
			totalPlatinum += float64(totalCurrency.CopperBank / copperMod)
			totalPlatinum += float64(totalCurrency.PlatinumCursor)
			totalPlatinum += float64(totalCurrency.GoldCursor / goldMod)
			totalPlatinum += float64(totalCurrency.SilverCursor / silverMod)
			totalPlatinum += float64(totalCurrency.CopperCursor / copperMod)

			metrics["platinum"] = int64(totalPlatinum)
			metrics["ebon"] = int64(totalCurrency.EbonCrystals)
			metrics["radiant"] = int64(totalCurrency.RadiantCrystals)
			err = sendMetrics("currency", "total", metrics)
			if err != nil {
				fmt.Println("failed to send card metrics", err.Error())
			}
			//log.Println(totalPlatinum)
			//currencyCount.WithLabelValues("normal").Observe(float64(totalPlatinum))
			//ebonCount.WithLabelValues("normal").Observe(float64(totalCurrency.EbonCrystals))
			//radiantCount.WithLabelValues("normal").Observe(float64(totalCurrency.RadiantCrystals))
			time.Sleep(60 * time.Second)
		}
	}()

	// Expose the registered metrics via HTTP.
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func sendMetrics(series string, name string, entries map[string]interface{}) (err error) {
	var resp *http.Response
	var data []byte

	msg := fmt.Sprintf("%s,name=\"%s\" ", series, name)
	for k, v := range entries {
		switch val := v.(type) {
		case int64:
			msg += fmt.Sprintf("%s=%d,", k, val)
		case string:
			msg += fmt.Sprintf("%s=\"%s\",", k, val)
		default:
			err = errors.Wrapf(err, "invalid type passed for %s", k)
			return
		}
	}
	msg = fmt.Sprintf("%s %d", msg[0:len(msg)-1], time.Now().Unix())

	creds := ""
	if iconfig.Influx.User != "" {
		creds = fmt.Sprintf("&u=%s&p=%s", iconfig.Influx.User, iconfig.Influx.Password)
	}

	buf := bytes.NewBufferString(msg)
	resp, err = http.Post(fmt.Sprintf("%s/write?db=%s%s&precision=s", iconfig.Influx.URL, iconfig.Influx.Database, creds), "binary", buf)
	if err != nil {
		err = errors.Wrap(err, "failed to post metrics")
		return
	}
	fmt.Println(msg)
	if resp.StatusCode != 204 {
		data, err = ioutil.ReadAll(resp.Body)
		fmt.Println("response from influx not 204:", resp.Status, string(data))
		return
	}
	return
}
