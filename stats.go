package main

import (
	"database/sql"
	"fmt"
	"github.com/dustin/go-humanize"
	//"log"
	"strings"
	"time"
)

type Item struct {
	full, short, long, name, keys, itype string
	zone, date                           string
	id, wt, val                          int
	slots, effs, flags, restrs, supps    [][]string
	attrs, specs, enchs, resis, procs    [][]string
	tdate                                time.Time
}

func (i *Item) FillItemByID(id int) { // replace with ORM?
	i.id = id

	db := OpenDB()
	defer db.Close()

	query := "select item_name, keywords, weight, c_value, item_type, " +
		"from_zone, short_stats, long_stats, last_id " +
		"from items where item_id = ?"
	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	err = stmt.QueryRow(i.id).Scan(
		&i.name, &i.keys, &i.wt, &i.val, &i.itype,
		&i.zone, &i.short, &i.long, &i.date,
	)
	ChkErr(err)
	stmt.Close()

	query = "select i.slot_abbr, worn_slot, slot_display " +
		"from item_slots i, slots s " +
		"where i.slot_abbr = s.slot_abbr and item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err := stmt.Query(i.id)
	ChkErr(err)
	defer rows.Close()

	// awkward copypasta, but very generic and repeatable
	cols, err := rows.Columns()
	ChkErr(err)
	pointers := make([]interface{}, len(cols))
	container := make([]string, len(cols))
	for i := range pointers {
		pointers[i] = &container[i]
	}

	for rows.Next() {
		err = rows.Scan(pointers...)
		i.slots = append(i.slots, container)
	}
	ChkRows(rows)
	stmt.Close()
}

var i struct {
	name, itype, zone, date, keys, s         string // base
	wt, val                                  int    // base
	specs, procs, enchs, flags, restr, zones string // temp
	tmp, tmpb, tmpc, txt                     string // temp
	tmp1, tmp2, tmp3, tmp4                   int    // temp
	txt1, txt2, txt3, txt4, txt5             string // temp, special ordering
}

func FormatStats() []string {
	t1 := time.Now()

	db := OpenDB()
	defer db.Close()

	query := "SELECT count(item_id) FROM items"
	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var size int
	err = stmt.QueryRow().Scan(&size)
	ChkErr(err)
	stmt.Close()

	// switch to multidimensional
	ids := make([]int, size, size)
	short := make([]string, size, size)
	long := make([]string, size, size)

	//log.Printf("len(ids) = %d\b", size)
	query = "SELECT item_id FROM items"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err := stmt.Query()
	ChkErr(err)
	defer rows.Close()

	counter := 0
	var id int
	for rows.Next() {
		err = rows.Scan(&id)
		ids[counter] = id
		//log.Printf("ids[%d] = %d\n", counter, id)
		counter++
	}
	ChkRows(rows)

	for n := range ids {
		short[n] = ConstructShortStats(db, ids[n])
		long[n] = ConstructLongStats(db, ids[n])
	}

	// put the batched short_stats into the database
	tx, err := db.Begin()
	ChkErr(err)
	stmt, err = tx.Prepare(
		"UPDATE items SET short_stats = ?, long_stats = ? " +
			"WHERE item_id = ?")
	ChkErr(err)
	defer stmt.Close()

	for n := range ids {
		_, err = stmt.Exec(short[n], long[n], ids[n])
		ChkErr(err)
	}

	tx.Commit()

	t2 := time.Now()
	txt := []string{fmt.Sprintf("Runtime: %v\n", t2.Sub(t1).String())}
	return txt
}

func ConstructShortStats(db *sql.DB, id int) string {
	// collect general item information
	query := "SELECT item_name, item_type, weight, c_value, " +
		"from_zone, CAST(last_id AS TEXT) " +
		"FROM items WHERE item_id = ?"
	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	err = stmt.QueryRow(id).Scan(
		&i.name, &i.itype, &i.wt, &i.val,
		&i.zone, &i.date,
	)
	ChkErr(err)
	stmt.Close()
	i.s = i.name

	// collect slots (i.slots, i.slot)
	query = "SELECT slot_abbr FROM item_slots WHERE item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err := stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	i.tmp1 = 0
	for rows.Next() {
		err = rows.Scan(&i.tmp)
		if i.tmp1 == 0 {
			i.txt = strings.Title(i.tmp)
		} else {
			i.txt += " or " + strings.Title(i.tmp)
		}
		i.tmp1++
	}
	ChkRows(rows)
	stmt.Close()
	if i.tmp1 > 0 {
		i.s += " (" + i.txt + ")"
	}

	// collect armor class (i.spec, but only for armor)
	query = "SELECT spec_value FROM item_specials " +
		"WHERE item_id = ? AND spec_abbr = 'ac'"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp1)
		i.s += fmt.Sprintf(" AC:%d", i.tmp1)
	}
	ChkRows(rows)
	stmt.Close()

	// collect attributes (i.attrs, i.attr)
	query = "SELECT attrib_abbr, attrib_value " +
		"FROM item_attribs WHERE item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp, &i.tmp1)
		i.s += fmt.Sprintf(" %s:%d", strings.Title(i.tmp), i.tmp1)
	}
	ChkRows(rows)
	stmt.Close()

	// collect resistances (i.resis, i.res)
	query = "SELECT resist_abbr, resist_value " +
		"FROM item_resists WHERE item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp, &i.tmp1)
		i.s += fmt.Sprintf(" %s:%d%%", strings.Title(i.tmp), i.tmp1)
	}
	ChkRows(rows)
	stmt.Close()

	// collect item effects (i.effs, i.eff)
	query = "SELECT effect_abbr FROM item_effects WHERE item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp)
		i.s += " " + strings.ToUpper(i.tmp)
	}
	ChkRows(rows)
	stmt.Close()

	// collect specials (i.specs, i.spec) and break them down by type
	i.specs = " *"
	i.txt1, i.txt2, i.txt3, i.txt4, i.txt5 = "", "", "", "", ""
	query = "SELECT item_type, spec_abbr, spec_value " +
		"FROM item_specials WHERE item_id = ? AND spec_abbr != 'ac'"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.txt, &i.tmp, &i.tmpb)
		if !strings.Contains(i.specs, "(") {
			i.specs += " (" + strings.Title(i.txt) + ")"
		}
		switch {
		case i.txt == "crystal" || i.txt == "spellbook" ||
			i.txt == "comp_bag" || i.txt == "ammo":
			i.txt1 += " " + strings.Title(i.tmp) + ":" + i.tmpb
		case i.txt == "container":
			if i.tmp == "holds" {
				i.txt1 += " Holds:" + i.tmpb
			} else if i.tmp == "wtless" {
				i.txt2 += " Wtless:" + i.tmpb
			}
		case i.txt == "poison":
			if i.tmp == "level" {
				i.txt1 += " Lvl:" + i.tmpb
			} else if i.tmp == "type" {
				i.txt2 += " Type:" + i.tmpb
			} else if i.tmp == "apps" {
				i.txt3 += " Apps:" + i.tmpb
			} else if i.tmp == "hits" {
				i.txt4 += " Hits:" + i.tmpb
			}
		case i.txt == "scroll" || i.txt == "potion":
			if i.tmp == "level" {
				i.txt1 += " Lvl:" + i.tmpb
			} else if i.tmp == "spell1" {
				i.txt2 += " " + i.tmpb
			} else if i.tmp == "spell2" {
				i.txt3 += " - " + i.tmpb
			} else if i.tmp == "spell3" {
				i.txt4 += " - " + i.tmpb
			}
		case i.txt == "staff" || i.txt == "wand":
			if i.tmp == "level" {
				i.txt1 += " Lvl:" + i.tmpb
			} else if i.tmp == "spell" {
				i.txt2 += " " + i.tmpb
			} else if i.tmp == "charges" {
				i.txt3 += " Charges:" + i.tmpb
			}
		case i.txt == "instrument":
			if i.tmp == "quality" {
				i.txt1 += " Quality:" + i.tmpb
			} else if i.tmp == "stutter" {
				i.txt2 += " Stutter:" + i.tmpb
			} else if i.tmp == "min_level" {
				i.txt3 += " Min_Level:" + i.tmpb
			} else if i.tmp == "type" {
				i.txt4 += " Type:" + i.tmpb
			}
		case i.txt == "weapon":
			if i.tmp == "dice" {
				i.txt1 += " Dice:" + i.tmpb
			} else if i.tmp == "crit" {
				i.txt2 += " Crit:" + i.tmpb + "%"
			} else if i.tmp == "multi" {
				i.txt3 += " Multi:" + i.tmpb + "x"
			} else if i.tmp == "class" {
				i.txt4 += " (Class: " + i.tmpb + ","
			} else if i.tmp == "type" {
				i.txt5 += " Type: " + i.tmpb + ")"
			}
		}
	}
	ChkRows(rows)
	stmt.Close()
	i.specs += i.txt1 + i.txt2 + i.txt3 + i.txt4 + i.txt5
	if i.specs != " *" {
		i.s += i.specs
	}

	// collect procs (i.procs, i.proc)
	i.procs = " *"
	query = "SELECT proc_name FROM item_procs WHERE item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp)
		if i.procs == " *" {
			i.procs += " Procs: " + i.tmp
		} else {
			i.procs += " - " + i.tmp
		}
	}
	ChkRows(rows)
	stmt.Close()
	if i.procs != " *" {
		i.s += i.procs
	}

	// collect enchantments (i.enchs, i.ench)
	i.enchs = " *"
	query = "SELECT ench_name, dam_pct, freq_pct, sv_mod, duration " +
		"FROM item_enchants WHERE item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp, &i.tmp1, &i.tmp2, &i.tmp3, &i.tmp4)
		if i.enchs != " *" {
			i.enchs += " -"
		}
		i.enchs += fmt.Sprintf(" %s %d%% %d%% %d %d",
			strings.Title(i.tmp), i.tmp1, i.tmp2, i.tmp3, i.tmp4)
	}
	ChkRows(rows)
	stmt.Close()
	if i.enchs != " *" {
		i.s += i.enchs
	}

	// collect item flags (i.flags, i.flag)
	i.flags = " *"
	query = "SELECT flag_abbr FROM item_flags WHERE item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp)
		if strings.Contains(i.tmp, "_") {
			s := strings.Split(i.tmp, "_")
			for n, v := range s {
				s[n] = strings.Title(v)
			}
			i.tmp = strings.Join(s, "_")
			i.flags += " " + i.tmp
		} else {
			i.flags += " " + strings.Title(i.tmp)
		}
	}
	ChkRows(rows)
	stmt.Close()
	if i.flags != " *" {
		i.s += i.flags
	}

	// collect restrictions (i.restr, i.rest)
	i.restr = " *"
	query = "SELECT restrict_abbr FROM item_restricts WHERE item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp)
		i.restr += " " + strings.Title(i.tmp)
	}
	ChkRows(rows)
	stmt.Close()
	if i.restr != " *" && i.flags == " *" {
		i.s += i.restr
	} else if i.restr != " *" && i.flags != " *" {
		i.restr = i.restr[:1] + i.restr[3:]
		i.s += i.restr
	}

	// collect item supplementals (i.supps, i.supp)
	query = "SELECT supp_value FROM supps s, item_supps i " +
		"WHERE s.supp_abbr = i.supp_abbr AND item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	// put in misc info
	i.itype = " *"
	if i.wt != -1 {
		i.itype += fmt.Sprintf(" Wt:%d", i.wt)
	}
	if i.val != -1 {
		i.val = i.val / 1000
		i.itype += fmt.Sprintf(" Val:%dp", i.val)
	}

	// construct the zone and last id
	i.zones = ""
	for rows.Next() {
		err = rows.Scan(&i.tmp)
		if i.tmp != "NoID" {
			i.zones += i.tmp
		} else {
			i.itype += " " + i.tmp
		}
	}
	ChkRows(rows)
	stmt.Close()
	if i.zones != "" {
		i.zone += " (" + i.zones + ")"
	}
	if i.itype != " *" {
		i.s += i.itype
	}
	i.s += " * Zone: " + i.zone + " * Last ID: " + i.date

	// debugging
	//log.Println(i.s)

	return i.s
}

func ConstructLongStats(db *sql.DB, id int) string {
	// collect general item information
	query := "SELECT item_name, item_type, weight, c_value, " +
		"zone_name, CAST(last_id AS TEXT), keywords " +
		"FROM items i, zones z " +
		"WHERE i.from_zone = z.zone_abbr AND item_id = ?"
	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	err = stmt.QueryRow(id).Scan(
		&i.name, &i.itype, &i.wt, &i.val,
		&i.zone, &i.date, &i.keys,
	)
	ChkErr(err)
	stmt.Close()
	i.s = i.name

	// collect slots (i.slots, i.slot)
	query = "SELECT slot_display " +
		"FROM item_slots i, slots s " +
		"WHERE i.slot_abbr = s.slot_abbr AND item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err := stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	i.tmp1 = 0
	for rows.Next() {
		err = rows.Scan(&i.tmp)
		if i.tmp1 == 0 {
			i.txt = i.tmp
		} else {
			i.txt += " or " + i.tmp
		}
		i.tmp1++
	}
	ChkRows(rows)
	stmt.Close()
	if i.tmp1 == 1 {
		i.s += " (Slot: " + i.txt + ")"
	} else if i.tmp1 > 1 {
		i.s += " (Slots: " + i.txt + ")"
	}

	// collect armor class (specials, but only for armor)
	query = "SELECT spec_display, spec_value " +
		"FROM item_specials i, specials s " +
		"WHERE i.spec_abbr = s.spec_abbr AND item_id = ? " +
		"AND i.spec_abbr = 'ac'"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp, &i.tmp1)
		i.s += fmt.Sprintf(" %s: %d", i.tmp, i.tmp1)
	}
	ChkRows(rows)
	stmt.Close()

	// collect attributes (i.attrs, i.attr)
	query = "SELECT attrib_display, attrib_value " +
		"FROM item_attribs i, attribs a " +
		"WHERE i.attrib_abbr = a.attrib_abbr AND item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp, &i.tmp1)
		i.s += fmt.Sprintf(", %s: %d", i.tmp, i.tmp1)
	}
	ChkRows(rows)
	stmt.Close()

	// collect resistances (i.resis, i.res)
	query = "SELECT resist_display, resist_value " +
		"FROM item_resists i, resists r " +
		"WHERE i.resist_abbr = r.resist_abbr AND item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp, &i.tmp1)
		i.s += fmt.Sprintf(", %s: %d%%", i.tmp, i.tmp1)
	}
	ChkRows(rows)
	stmt.Close()

	// collect item effects (i.effs, i.eff)
	query = "SELECT effect_display " +
		"FROM item_effects i, effects e " +
		"WHERE i.effect_abbr = e.effect_abbr AND item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp)
		i.s += ", " + i.tmp
	}
	ChkRows(rows)
	stmt.Close()

	// collect specials (i.specs, i.spec) and break them down by type
	i.specs = " *"
	i.txt1, i.txt2, i.txt3, i.txt4, i.txt5 = "", "", "", "", ""
	query = "SELECT i.item_type, i.spec_abbr, spec_value, spec_display " +
		"FROM item_specials i, specials s " +
		"WHERE i.spec_abbr = s.spec_abbr " +
		"AND i.item_type = s.item_type AND item_id = ? " +
		"AND i.spec_abbr != 'ac'"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.txt, &i.tmp, &i.tmpb, &i.tmpc)
		switch {
		case i.txt == "crystal" || i.txt == "spellbook" ||
			i.txt == "comp_bag" || i.txt == "ammo":
			i.txt1 += " " + i.tmpc + ": " + i.tmpb
		case i.txt == "container":
			if i.tmp == "holds" {
				i.txt1 += " " + i.tmpc + ": " + i.tmpb
			} else if i.tmp == "wtless" {
				i.txt2 += ", " + i.tmpc + ": " + i.tmpb
			}
		case i.txt == "poison":
			if i.tmp == "level" {
				i.txt1 += " " + i.tmpc + ": " + i.tmpb
			} else if i.tmp == "type" {
				i.txt2 += ", " + i.tmpc + ": " + i.tmpb
			} else if i.tmp == "apps" {
				i.txt3 += ", " + i.tmpc + ": " + i.tmpb
			} else if i.tmp == "hits" {
				i.txt4 += ", " + i.tmpc + ": " + i.tmpb
			}
		case i.txt == "scroll" || i.txt == "potion":
			if i.tmp == "level" {
				i.txt1 += " " + i.tmpc + ": " + i.tmpb
			} else if i.tmp == "spell1" {
				i.txt2 += " " + i.tmpb
			} else if i.tmp == "spell2" {
				i.txt3 += " - " + i.tmpb
			} else if i.tmp == "spell3" {
				i.txt4 += " - " + i.tmpb
			}
		case i.txt == "staff" || i.txt == "wand":
			if i.tmp == "level" {
				i.txt1 += " " + i.tmpc + ": " + i.tmpb
			} else if i.tmp == "spell" {
				i.txt2 += " " + i.tmpb
			} else if i.tmp == "charges" {
				i.txt3 += " " + i.tmpc + ": " + i.tmpb
			}
		case i.txt == "instrument":
			if i.tmp == "quality" {
				i.txt1 += " " + i.tmpc + ": " + i.tmpb
			} else if i.tmp == "stutter" {
				i.txt2 += ", " + i.tmpc + ": " + i.tmpb
			} else if i.tmp == "min_level" {
				i.txt3 += ", " + i.tmpc + ": " + i.tmpb
			} else if i.tmp == "type" {
				i.txt4 += ", " + i.tmpc + ": " + i.tmpb
			}
		case i.txt == "weapon":
			if i.tmp == "dice" {
				i.txt1 += " " + i.tmpc + ": " + i.tmpb
			} else if i.tmp == "crit" {
				i.txt2 += ", " + i.tmpc + ": " + i.tmpb + "%"
			} else if i.tmp == "multi" {
				i.txt3 += ", " + i.tmpc + ": " + i.tmpb + "x"
			} else if i.tmp == "class" {
				i.txt4 += ", (" + i.tmpc + ": " + i.tmpb
			} else if i.tmp == "type" {
				i.txt5 += ", " + i.tmpc + ": " + i.tmpb + ")"
			}
		}
	}
	ChkRows(rows)
	stmt.Close()
	i.specs += i.txt1 + i.txt2 + i.txt3 + i.txt4 + i.txt5
	if i.specs != " *" {
		i.s += " (Item Type: " + strings.Title(i.itype) + ")" +
			i.specs[:1] + i.specs[3:]
	} else {
		i.s += " (Item Type: " + strings.Title(i.itype) + ")"
	}

	// collect procs (i.procs, i.proc)
	i.procs = " *"
	query = "SELECT proc_name FROM item_procs WHERE item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp)
		if i.procs == " *" {
			i.procs += " Procs: " + i.tmp
		} else {
			i.procs += " - " + i.tmp
		}
	}
	ChkRows(rows)
	stmt.Close()
	if i.procs != " *" {
		i.s += i.procs
	}

	// collect enchantments (i.enchs, i.ench)
	i.enchs = " *"
	query = "SELECT ench_name, dam_pct, freq_pct, sv_mod, duration " +
		"FROM item_enchants WHERE item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp, &i.tmp1, &i.tmp2, &i.tmp3, &i.tmp4)
		if i.enchs != " *" {
			i.enchs += " -"
		}
		i.enchs += fmt.Sprintf(" %s %d%% %d%% %d %d",
			strings.Title(i.tmp), i.tmp1, i.tmp2, i.tmp3, i.tmp4)
	}
	ChkRows(rows)
	stmt.Close()
	if i.enchs != " *" {
		i.s += i.enchs
	}

	// collect item flags (i.flags, i.flag)
	i.flags = " *"
	query = "SELECT flag_display " +
		"FROM item_flags i, flags f " +
		"WHERE i.flag_abbr = f.flag_abbr AND item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp)
		i.flags += ", " + i.tmp
	}
	ChkRows(rows)
	stmt.Close()
	if i.flags != " *" {
		i.s += i.flags
	}

	// collect restrictions (i.restr, i.rest)
	i.restr = " *"
	query = "SELECT restrict_name " +
		"FROM item_restricts i, restricts r " +
		"WHERE i.restrict_abbr = r.restrict_abbr AND item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.tmp)
		i.restr += " " + strings.Title(i.tmp)
	}
	ChkRows(rows)
	stmt.Close()
	if i.restr != " *" && i.flags == " *" {
		i.s += i.restr
	} else if i.restr != " *" && i.flags != " *" {
		i.restr = i.restr[:1] + i.restr[3:]
		i.s += i.restr
	}

	// throw keywords on there
	i.s += " * Keywords:(" + i.keys + ")"

	// collect item supplementals (i.supps, i.supp)
	query = "SELECT supp_display " +
		"FROM item_supps i, supps s " +
		"WHERE i.supp_abbr = s.supp_abbr AND item_id = ?"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err = stmt.Query(id)
	ChkErr(err)
	defer rows.Close()

	// put in misc info
	i.itype = " *"
	if i.wt != -1 {
		i.itype += fmt.Sprintf(" Weight: %d", i.wt)
	}
	if i.val != -1 {
		i.itype += fmt.Sprintf(
			", Value: %s copper",
			humanize.Comma(int64(i.val)),
		)
	}

	// construct the zone and last id
	i.zones = ""
	for rows.Next() {
		err = rows.Scan(&i.tmp)
		if i.tmp != "No Identify" {
			i.zones += ", " + i.tmp
		} else {
			i.itype += ", " + i.tmp
		}
	}
	ChkRows(rows)
	stmt.Close()
	if i.zones != "" {
		i.zone += " (" + i.zones + ")"
	}
	if i.itype != " *" {
		i.s += i.itype
	}
	i.s += " * Zone: " + i.zone + " * Last ID: " + i.date

	i.s = strings.Replace(i.s, "(, ", "(", -1)
	i.s = strings.Replace(i.s, "*, ", "* ", -1)
	// debugging
	//log.Println(i.s)

	return i.s
}
