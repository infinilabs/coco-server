package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

func main() {
	dir := os.Args[1]
	// arg 2 ("truncate") kept for CLI compatibility — badger v4 removed the
	// option and always replays the WAL on open
	_ = os.Args[2] == "truncate"
	opts := badger.DefaultOptions(dir).
		WithReadOnly(false).
		WithLogger(nil).
		WithCompression(options.None).
		WithNumGoroutines(1)
	db, err := badger.Open(opts)
	if err != nil {
		fmt.Println("open error:", err)
		os.Exit(1)
	}
	defer db.Close()

	prefix := os.Args[3]
	want := map[string]bool{}
	for _, id := range os.Args[4:] {
		want[id] = false
	}
	count := 0
	err = db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			key := string(item.Key())
			count++
			for id := range want {
				if strings.HasSuffix(key, id) {
					want[id] = true
					fmt.Printf("FOUND key=%s size=%d\n", key, item.ValueSize())
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("view error:", err)
	}
	fmt.Printf("total keys with prefix %q: %d\n", prefix, count)
	for id, found := range want {
		if !found {
			fmt.Printf("MISSING: %s\n", id)
		}
	}
}
