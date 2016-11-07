package tracking_station

import "github.com/HouzuoGuo/tiedot/db"

var DB *db.DB

func SetupDatabase() {
	var err error
	DB, _ = db.OpenDB("tsdb")
	if err != nil {
		panic(err)
	}

	cols := []string{"clients", "tracking"}

	// Create collections
	for _, col := range cols {
		notExist := (func(ac []string) bool {
			for _, c := range ac {
				if c == col {
					return false
				}
			}
			return true
		})(DB.AllCols())

		if notExist {
			if err := DB.Create(col); err != nil {
				panic(err)
			}
		}

		useCol := DB.Use(col)

		// Create indexes
		if col == "clients" {
			useCol.Index([]string{"client_id"})
			useCol.Index([]string{"client_id", "tags, part"})
			useCol.Index([]string{"client_id", "tags, device"})
			useCol.Index([]string{"client_id", "tags, url"})
		} else if col == "tracking" {
			useCol.Index([]string{"client_id"})
			useCol.Index([]string{"status"})
			useCol.Index([]string{"start_unixminute"})
			useCol.Index([]string{"end_unixminiute"})
		}
	}
}
