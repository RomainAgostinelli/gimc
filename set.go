package gimc

import (
	"github.com/RomainAgostinelli/gimc/pkg/heap"
	"io"
	"log"
)

const (
	DELETED       = 1 << 0
	ADDED         = 1 << 1
	MODIFIED      = 1 << 2
	ADDRESSLENGTH = uint8(32)
)

type set struct {
	ways  map[uint32][]byte // First byte in the array are tags
	cache *Cache            // Pointer to the cache used for shared options
	rePol repol
}

// createSet create a logical set of a cache
// tagSize is the size (number of bit) of the tag
// waysMax number of ways for the set, min 1
// dataSize size of the data
// blockSize is the size (number of bytes) to store in a cache entry, must be a power of 2
func createSet(cache *Cache) *set {
	s := &set{
		ways:  make(map[uint32][]byte),
		cache: cache,
	}
	switch cache.repol {
	case FIFO:
		s.rePol = &fifo{}
	case LRU:
		s.rePol = &lru{
			heap:    heap.NewHeap(int(cache.maxWays)),
			lclock:  0,
			maxSize: s.cache.maxWays,
		}
	default:
		log.Fatalln("Not known replacement policy.")
	}
	return s
}

func (s *set) get(address uint32) []byte {
	// get the tag
	tag := address >> (ADDRESSLENGTH - s.cache.tagSize)
	offset := address & s.cache.offsetMask
	var val []byte
	val, ok := s.ways[tag]
	if !ok {
		s.cache.missCount++
		// replacement policy
		s.replace(tag, address & ^s.cache.offsetMask)
		val = s.ways[tag]
		s.rePol.miss(tag)
	} else {
		s.cache.hitCount++
		s.rePol.hit(tag)
	}
	return val[offset+1 : uint32(s.cache.dataSize)+offset+1] // first byte are tags
}

func (s *set) replace(tag, address uint32) {
	if len(s.ways) >= int(s.cache.maxWays) { // all ways are full, remove the oldest one
		// Get the tag to replace
		toReplace := s.rePol.toReplace()
		s.ways[toReplace] = nil   // delete array
		delete(s.ways, toReplace) // delete entry
	}
	// create new tags and data
	val := make([]byte, s.cache.blockSize+1) // one for edition bits
	val[0] = 0b0000_0000                     // edition bits TODO: implement it
	n, err := s.cache.source.ReadAt(val[1:], int64(address))
	if err != nil || n < len(val[1:]) {
		if err == io.EOF {
			copy(val[1:], "EOF") // consider EOF
		} else {
			log.Fatalf("Read from file failed: %s, %b", err, n)
		}
	}
	// put ourself into the way
	s.ways[tag] = val
}

// repol interface represent the capabilities of a replacement policy implementation
type repol interface {
	// hit is called by a set when something has been found in the cache
	hit(tag uint32)
	// miss is called by a set when something was missing in the cache
	miss(tag uint32)
	// toReplace gives the tag present in the cache to replace
	toReplace() uint32
}
