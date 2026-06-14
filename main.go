package main

import (
	"os"

	_ "modernc.org/sqlite"

	"github.com/Sam-Blundell/messageboard/post"
)

type Test struct {
	id   int64
	body string
}

func main() {

	// db, err := sql.Open("sqlite", "database")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// err = db.Ping()
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// _, err = db.Exec("CREATE TABLE IF NOT EXISTS test (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, body TEXT)")
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// // result, err := db.Exec("INSERT INTO test (BODY) VALUES (\"FIRST\"), (\"SECOND\"), (\"THREE GET\")")
	// // if err != nil {
	// // 	fmt.Println(err)
	// // }
	// // fmt.Println(result)

	// var testRow Test
	// row := db.QueryRow("SELECT id, body FROM test LIMIT 1 OFFSET 99")
	// err = row.Scan(&testRow.id, &testRow.body)
	// if err != nil {
	// 	if errors.Is(err, sql.ErrNoRows) {

	// 	}
	// 	fmt.Println(err)
	// } else {
	// 	fmt.Println(testRow)
	// }

	store := post.NewFile("./posts.json")
	c := &cli{
		store:  store,
		in:     os.Stdin,
		out:    os.Stdout,
		errOut: os.Stderr,
	}
	c.run()
}
