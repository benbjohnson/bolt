package bolt_test

import (
	"encoding/binary"
	"hash/fnv"
	"sync"
	"testing"

	"github.com/boltdb/bolt"
)

func BenchmarkDBBatchAutomatic(b *testing.B) {
	db := NewTestDB()
	defer db.Close()
	db.MustCreateBucket([]byte("bench"))

	h := fnv.New32a()
	buf := make([]byte, 4)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := make(chan struct{})
		var wg sync.WaitGroup

		for round := 0; round < 1000; round++ {
			wg.Add(1)

			go func(id uint32) {
				defer wg.Done()
				<-start

				binary.LittleEndian.PutUint32(buf, id)
				h.Write(buf[:])
				k := h.Sum(nil)
				insert := func(tx *bolt.Tx) error {
					b := tx.Bucket([]byte("bench"))
					return b.Put(k, []byte("filler"))
				}
				if err := db.Batch(insert); err != nil {
					b.Error(err)
					return
				}
			}(uint32(round))
		}
		close(start)
		wg.Wait()
	}
	b.StopTimer()
}

func BenchmarkDBBatchSingle(b *testing.B) {
	db := NewTestDB()
	defer db.Close()
	db.MustCreateBucket([]byte("bench"))
	h := fnv.New32a()
	buf := make([]byte, 4)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := make(chan struct{})
		var wg sync.WaitGroup

		for round := 0; round < 1000; round++ {
			wg.Add(1)
			go func(id uint32) {
				defer wg.Done()
				<-start

				binary.LittleEndian.PutUint32(buf, id)
				h.Write(buf[:])
				k := h.Sum(nil)
				insert := func(tx *bolt.Tx) error {
					b := tx.Bucket([]byte("bench"))
					return b.Put(k, []byte("filler"))
				}
				if err := db.Update(insert); err != nil {
					b.Error(err)
					return
				}
			}(uint32(round))
		}
		close(start)
		wg.Wait()
	}
	b.StopTimer()
}

func BenchmarkDBBatchManual10x100(b *testing.B) {
	db := NewTestDB()
	defer db.Close()
	db.MustCreateBucket([]byte("bench"))
	h := fnv.New32a()
	buf := make([]byte, 4)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := make(chan struct{})
		var wg sync.WaitGroup

		for major := 0; major < 10; major++ {
			wg.Add(1)
			go func(id uint32) {
				defer wg.Done()
				<-start

				insert100 := func(tx *bolt.Tx) error {
					for minor := uint32(0); minor < 100; minor++ {
						binary.LittleEndian.PutUint32(buf, uint32(id*100+minor))
						h.Write(buf[:])
						k := h.Sum(nil)
						b := tx.Bucket([]byte("bench"))
						return b.Put(k, []byte("filler"))
					}
					return nil
				}
				if err := db.Update(insert100); err != nil {
					b.Fatal(err)
				}
			}(uint32(major))
		}
		close(start)
		wg.Wait()
	}

	b.StopTimer()
}
