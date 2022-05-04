package gimc

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/ag0st/bst"
	"log"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"testing"
	"time"
)

var keepCache *Cache
var keepMMU *bst.BST

type HashList [][32]byte

func TestCreateData(t *testing.T) {
	file, err := os.Create("hashes.txt")
	defer file.Close()
	if err != nil {
		log.Fatalln("Cannot create file")
	}
	for i := 0; i < 2_000_000; i++ {
		sum256 := sha256.Sum256([]byte(strconv.Itoa(i)))
		_, err = file.Write(sum256[:])
		if err != nil {
			log.Fatalln("Error while writing")
		}
	}

}

func TestGet(t *testing.T) {
	fd := NewFileDatasource("hashes.txt")
	cache, err := CreateCache(1024, 4096, 32, 1, fd, FIFO)
	if err != nil {
		t.Fatal(fmt.Sprintf("Cannot create cache: %s", err))
	}
	defer func(cache *Cache) {
		err := cache.Close()
		if err != nil {
			t.Fatal(fmt.Sprintf("Cannot close cache: %s", err))
		}
	}(cache)
	for i := 0; i < 5_000_000; i++ {
		random := rand.Intn(2_000_000)
		// conversion ok, max 2 mio
		sum256 := sha256.Sum256([]byte(strconv.Itoa(random)))
		get := cache.Get(uint32(random) * 32)
		if bytes.Compare(get, sum256[:]) != 0 {
			t.Fatal("Not same sha")
		}

	}
}

func BenchmarkGetWithCache(b *testing.B) {
	fd := NewFileDatasource("hashes.txt")
	keepCache, _ = CreateCache(512, 4096, 32, 1, fd, FIFO)
	defer keepCache.Close()
	keepCache.ResetCounters()
	b.Run(
		"Continuous access with cache", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				keepCache.Get(uint32(i%2_000_000) * 32) // data are 32 bytes long
			}
		},
	)
	// display cache metrics
	hits, misses := keepCache.GetCounters()
	log.Printf("Total misses %d", misses)
	log.Printf("Total hits %d", hits)
	log.Printf("Hits/Misses ration %d/%d = %f", hits, misses, float64(hits)/float64(misses))
}

func BenchmarkGetWithoutCache(b *testing.B) {
	fd := NewFileDatasource("hashes.txt")
	fd.Open()
	defer fd.Close()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var data [32]byte // 32 bytes data
		fd.ReadAt(data[:], int64(i%2_000_000)*32)
	}
}

func BenchmarkGetWithoutCache2(b *testing.B) {
	file, _ := os.Open("hashes.txt")
	defer file.Close()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var data [32]byte // 32 bytes data
		file.ReadAt(data[:], int64(i%2_000_000)*32)
	}
}

// Test with hashes:
// 1. Create file with sorted hashes (create BST data also)
// 2. Convert a hash into an address (32 bits prefix)
// 3. Ask the cache for this address  --> May broke as it does not exists
// 4. Compare the hashes.
func BenchmarkGetHashes(b *testing.B) {
	// 1.
	hashNb := 15_000_000
	var sets, blockSize, dataSize, maxWays uint16 = 512, 512, 32, 12
	// Get back all prefixes for creating mmu
	allPrefix := setupTestHash(hashNb)
	// Create the datasource
	fd := NewFileDatasource("hashes-sorted.txt")
	// Create the cache
	var err error
	keepCache, err = CreateCache(sets, blockSize, dataSize, maxWays, fd, FIFO)
	if err != nil {
		b.Fatal(fmt.Sprintf("Error when creating the cache: %s", err))
	}
	// Create MMU
	keepMMU = bst.NewBSTReady(allPrefix[:])

	// Fulfill the cache before start to avoid cold start
	fulfillCache(hashNb, dataSize)

	// TESTING
	log.Println("MMU Built, Testing...")
	fileCpu, _ := os.Create("cpu.pprof")
	defer fileCpu.Close()

	fileMem, _ := os.Create("mem.pprof")
	defer fileMem.Close()

	if err := pprof.StartCPUProfile(fileCpu); err != nil {
		log.Fatalln("Cannot write cpu profiling")
	}
	defer pprof.StopCPUProfile()

	b.Run(
		"TEST PRESENT", func(b *testing.B) {

			keepCache.ResetCounters()

			// With cache
			rand.Seed(time.Now().UnixNano())
			b.Run(
				"WITH CACHE", func(b *testing.B) {
					benchmarkPresentCache(b, hashNb, dataSize)
				},
			)

			// Without cache
			rand.Seed(time.Now().UnixNano())
			b.Run(
				"WITHOUT CACHE", func(b *testing.B) {
					benchmarkPresentNoCache(b, hashNb, dataSize, fd)
				},
			)
		},
	)

	b.Run(
		"TEST NOT PRESENT", func(b *testing.B) {

			keepCache.ResetCounters()

			// With cache
			rand.Seed(time.Now().UnixNano())
			b.Run(
				"WITH CACHE", func(b *testing.B) {
					benchmarkNotPresentCache(b, hashNb, dataSize)
				},
			)

			// Without cache
			rand.Seed(time.Now().UnixNano())
			b.Run(
				"WITHOUT CACHE", func(b *testing.B) {
					benchmarkNotPresentNoCache(b, hashNb, dataSize, fd)
				},
			)
		},
	)

	runtime.GC()
	if err := pprof.WriteHeapProfile(fileMem); err != nil {
		log.Fatalln("Cannot write heap profiling")
	}
}

func benchmarkPresentCache(b *testing.B, hashNb int, dataSize uint16) {
	var totalRandomAccess uint64
	var totalIter uint64
	b.ReportAllocs()
	b.N = 5_000_000
	for i := 0; i < b.N; i++ {
		if i%500_000 == 0 {
			log.Printf("%d iterations done", totalIter)
		}
		totalIter++
		random := rand.Intn(hashNb)
		data := []byte(strconv.Itoa(random))

		sum256 := sha256.Sum256(data)
		_, ele, succ := keepMMU.GetPredSucc(
			&BSTEntry{
				diskPosition: 0,
				addr:         toAddress(sum256),
				// Loop to check until the next prefix
			},
		)
		upperBound := uint32(hashNb * 32)
		if succ != nil {
			upperBound = succ.(*BSTEntry).diskPosition
		}
		for j := ele.(*BSTEntry).diskPosition; j < upperBound; j += uint32(dataSize) {
			totalRandomAccess++
			res := keepCache.Get(j)
			if bytes.Compare(res[:], sum256[:]) == 0 {
				break
			}
		}
	}
	// display cache metrics
	hits, misses := keepCache.GetCounters()
	log.Printf("Total misses %d", misses)
	log.Printf("Total hits %d", hits)
	log.Printf("Hits %% : %f%%", (float64(hits)*100.0)/float64(hits+misses))
	log.Printf("Total Cache calls: %d", totalRandomAccess)
	log.Printf("Average Cache calls: %f", float64(totalRandomAccess)/float64(totalIter))
}

func benchmarkPresentNoCache(b *testing.B, hashNb int, dataSize uint16, fd Datasource) {
	var totalRandomAccess uint64
	var totalIter uint64
	b.ReportAllocs()
	val := make([]byte, 32)
	b.N = 5_000_000
	for i := 0; i < b.N; i++ {
		if i%500_000 == 0 {
			log.Printf("%d iterations done", totalIter)
		}
		totalIter++
		random := rand.Intn(hashNb)
		data := []byte(strconv.Itoa(random))

		sum256 := sha256.Sum256(data)

		_, ele, succ := keepMMU.GetPredSucc(
			&BSTEntry{
				diskPosition: 0,
				addr:         toAddress(sum256),
				// Loop to check until the next prefix
			},
		)
		upperBound := uint32(hashNb) * uint32(dataSize)
		if succ != nil {
			upperBound = succ.(*BSTEntry).diskPosition
		}

		for j := ele.(*BSTEntry).diskPosition; j < upperBound; j += uint32(dataSize) {
			totalRandomAccess++
			_, _ = fd.ReadAt(val, int64(j))
			if bytes.Compare(val[:], sum256[:]) == 0 {
				break
			}
		}
	}
	// TEAR-DOWN
	log.Printf("Total Random Access: %d", totalRandomAccess)
	log.Printf("Average Random Access: %f", float64(totalRandomAccess)/float64(totalIter))
}

func benchmarkNotPresentCache(b *testing.B, hashNb int, dataSize uint16) {
	var totalTests uint64
	var totalRandomAccess uint64
	var totalIter uint64
	b.ReportAllocs()
	b.N = 5_000_000
	for i := 0; i < b.N; i++ {
		if i%500_000 == 0 {
			log.Printf("%d iterations done", totalIter)
		}
		totalIter++
		random := rand.Intn(hashNb) + hashNb
		data := []byte(strconv.Itoa(random))

		sum256 := sha256.Sum256(data)
		_, ele, succ := keepMMU.GetPredSucc(
			&BSTEntry{
				diskPosition: 0,
				addr:         toAddress(sum256),
				// Loop to check until the next prefix
			},
		)
		if ele == nil {
			// prefix not present
			continue
		}
		totalTests++
		upperBound := uint32(hashNb * 32)
		if succ != nil {
			upperBound = succ.(*BSTEntry).diskPosition
		}
		for j := ele.(*BSTEntry).diskPosition; j < upperBound; j += uint32(dataSize) {
			totalRandomAccess++
			_ = keepCache.Get(j)
		}
	}
	// display cache metrics
	hits, misses := keepCache.GetCounters()
	log.Printf("Total misses %d", misses)
	log.Printf("Total hits %d", hits)
	log.Printf("Hits %% : %f%%", (float64(hits)*100.0)/float64(hits+misses))
	log.Printf("Total Cache calls: %d", totalRandomAccess)
	log.Printf("Average Cache calls: %f", float64(totalRandomAccess)/float64(totalTests))
}
func benchmarkNotPresentNoCache(b *testing.B, hashNb int, dataSize uint16, fd Datasource) {
	var totalRandomAccess uint64
	var totalIter uint64
	var totalTests uint64
	b.ReportAllocs()
	val := make([]byte, 32)
	b.N = 5_000_000
	for i := 0; i < b.N; i++ {
		if i%500_000 == 0 {
			log.Printf("%d iterations done", totalIter)
		}
		totalIter++
		random := rand.Intn(hashNb) + hashNb
		data := []byte(strconv.Itoa(random))
		sum256 := sha256.Sum256(data)
		_, ele, succ := keepMMU.GetPredSucc(
			&BSTEntry{
				diskPosition: 0,
				addr:         toAddress(sum256),
			},
		)
		if ele == nil {
			// prefix not present
			continue
		}
		totalTests++
		upperBound := uint32(hashNb * 32)
		if succ != nil {
			upperBound = succ.(*BSTEntry).diskPosition
		}
		for j := ele.(*BSTEntry).diskPosition; j < upperBound; j += uint32(dataSize) {
			totalRandomAccess++
			_, _ = fd.ReadAt(val, int64(j))
		}
	}
	// TEAR-DOWN
	log.Printf("Total Random Access: %d", totalRandomAccess)
	log.Printf("Average Random Access: %f", float64(totalRandomAccess)/float64(totalTests))
}

func fulfillCache(hashNb int, dataSize uint16) {
	var totalIter uint64

	for i := 0; i < 5_000_000; i++ {
		if i%500_000 == 0 {
			log.Printf("%d iterations done", totalIter)
		}
		totalIter++
		random := rand.Intn(hashNb) + hashNb
		data := []byte(strconv.Itoa(random))

		sum256 := sha256.Sum256(data)
		_, ele, succ := keepMMU.GetPredSucc(
			&BSTEntry{
				diskPosition: 0,
				addr:         toAddress(sum256),
				// Loop to check until the next prefix
			},
		)
		if ele == nil {
			// prefix not present
			continue
		}
		upperBound := uint32(hashNb * 32)
		if succ != nil {
			upperBound = succ.(*BSTEntry).diskPosition
		}
		for j := ele.(*BSTEntry).diskPosition; j < upperBound; j += uint32(dataSize) {
			_ = keepCache.Get(j)
		}
	}
}

func setupTestHash(hashNb int) []bst.Comparable {
	file, err := os.Create("hashes-sorted.txt")
	defer file.Close()
	if err != nil {
		log.Fatalln("Cannot create file")
	}
	var allHashes HashList
	for i := 0; i < hashNb; i++ {
		hash := sha256.Sum256([]byte(strconv.Itoa(i)))
		allHashes = append(allHashes, hash)
	}
	log.Println("Hashes generated")
	// sort
	sort.Sort(allHashes)
	log.Println("Hashes Sorted")
	// Store the address of each prefix
	var allPrefix []bst.Comparable
	for idx, hash := range allHashes {
		if idx%1_000_000 == 0 && idx > 0 {
			log.Printf("%d Hashes treated", idx)
		}
		_, err = file.Write(hash[:])
		if err != nil {
			log.Fatalln("Error while writing")
		}
		if idx == 0 || toAddress(hash) != toAddress(allHashes[idx-1]) { // not same first 4 elements
			// Add to all prefix
			allPrefix = append(
				allPrefix, &BSTEntry{
					diskPosition: uint32(idx) * 32, // number of the byte
					addr:         toAddress(hash),
				},
			)
		}
	}
	log.Println("Hashes Written")
	return allPrefix
}

type BSTEntry struct {
	diskPosition uint32
	addr         uint32
}

func (e *BSTEntry) CompareTo(other bst.Comparable) int {
	switch v := other.(type) {
	case *BSTEntry:
		if e.addr < v.addr {
			return -1
		} else if e.addr > v.addr {
			return 1
		} else {
			return 0
		}
	default:
		return -1
	}
}

func toAddress(hash [32]byte) uint32 {
	// take the 20 first bits as address (prefix)
	// return uint32(hash[0])<<(8*3) + uint32(hash[1])<<(8*2) + uint32(hash[2])<<(8*1) //+ uint32(hash[3])<<(8*0)
	return uint32(hash[0])<<(8*3) + uint32(hash[1])<<(8*2) + (uint32(hash[2])&uint32(0b11110000))<<(8*1) // + uint32(hash[3])<<(8*0)
}

func (h HashList) Len() int           { return len(h) }
func (h HashList) Less(i, j int) bool { return bytes.Compare(h[i][:], h[j][:]) < 0 }
func (h HashList) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
