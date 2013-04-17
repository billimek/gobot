package main

import (
	"flag"
	"log"
	"os/exec"
)

func main() {
	// for who.go Who(char, lvl)
	var char = flag.String("char", "", "Character name for update or import. Ex: Rynshana")
	var lvl = flag.Int("lvl", 0, "Character level for update or import. Ex: 50")
	// for who.go WhoChar(char, lvl, class, race, acct)
	var class = flag.String("class", "", "Character class for initial import. Ex: \"Cleric\"")
	var race = flag.String("race", "", "Character race for initial import. Ex: \"Moon Elf\"")
	var acct = flag.String("acct", "", "Character account for initial import. Ex: Krimic")
	// for identify.go Identify(filename)
	var file = flag.String("import", "", "Parse file for identify stats, import to DB. Ex: newstats.txt")
	// for time.go Uptime(curup)
	var time = flag.String("time", "", "Parse uptime for boot tracking. Ex: 58:10:26")
	// for tell.go ReplyTo(char, tell)
	var tell = flag.String("tell", "", "Tell with command and maybe operant. Ex: \"stat a longsword\"")
	// for who.go WhoBatch(ppl)
	var who = flag.String("who", "", "Batched who output. Ex: [ 1 Ctr] Rarac  (Orc)|[ 2 War] Xatus  (Troll)")
	// run database backup
	var bak = flag.Bool("backup", false, "Backup the toril.db database.")

	flag.Parse()

	// only run one command at a time
	switch {
	case *char != "" && 50 >= *lvl && *lvl > 0 && *class != "" && *race != "" && *acct != "":
		WhoChar(*char, *lvl, *class, *race, *acct)
	case *char != "" && 50 >= *lvl && *lvl > 0:
		Who(*char, *lvl)
	case *file != "":
		Identify(*file)
	case *time != "":
		Uptime(*time)
	case *char != "" && *tell != "":
		ReplyTo(*char, *tell)
	case *who != "":
		WhoBatch(*who)
	case *bak:
		cmd1 := exec.Command("sh", "-c", "echo '.dump' | sqlite3 toril.db | gzip -c >toril.db.`date +\"%Y-%m-%d\"`.gz")
		err := cmd1.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}
