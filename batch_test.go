package bolt_test

import (
	"testing"

	"github.com/boltdb/bolt"
)

// Ensure two functions can perform updates in a single batch.
func TestDB_Batch(t *testing.T) {
	db := NewTestDB()
	defer db.Close()
	db.MustCreateBucket([]byte("widgets"))

	// Iterate over multiple updates in separate goroutines.
	n := 2
	ch := make(chan error)
	for i := 0; i < n; i++ {
		go func(i int) {
			ch <- db.Batch(func(tx *bolt.Tx) error {
				return tx.Bucket([]byte("widgets")).Put(ui64tob(uint64(i)), []byte{})
			})
		}(i)
	}

	// Check all responses to make sure there's no error.
	for i := 0; i < n; i++ {
		if err := <-ch; err != nil {
			t.Fatal(err)
		}
	}

	// Ensure data is correct.
	db.MustView(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("widgets"))
		for i := 0; i < n; i++ {
			if v := b.Get(ui64tob(uint64(i))); v == nil {
				t.Errorf("key not found: %d", i)
			}
		}
		return nil
	})
}

func TestDB_Batch_Panic(t *testing.T) {
	db := NewTestDB()
	defer db.Close()

	var sentinel int
	var bork = &sentinel
	var problem interface{}
	var err error

	// Execute a function inside a batch that panics.
	func() {
		defer func() {
			if p := recover(); p != nil {
				problem = p
			}
		}()
		err = db.Batch(func(tx *bolt.Tx) error {
			panic(bork)
		})
	}()

	// Verify there is no error.
	if g, e := err, error(nil); g != e {
		t.Fatalf("wrong error: %v != %v", g, e)
	}
	// Verify the panic was captured.
	if g, e := problem, bork; g != e {
		t.Fatalf("wrong error: %v != %v", g, e)
	}
}
