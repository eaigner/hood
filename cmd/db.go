package main

func db(cmd string, args []string) {
	switch cmd {
	case "migrate":
		dbMigrate()
	}
}

func dbMigrate() {
	// TODO: implement
}
