package gimc

import (
	"errors"
	"fmt"
	"math"
)

// RePol is the type defining Replacement Policies for the cache
type RePol int

const (
	FIFO RePol = iota // FIFO = First In First Out
	LRU               // LRU  = Least Recently Used
)

type Datasource interface {
	ReadAt(p []byte, off int64) (n int, err error)
	WriteAt(p []byte, off int64) (n int, err error)
	Open() error
	Close() error
}

type Cache struct {
	sets                  []*set
	indexMask, offsetMask uint32 // Up to 16 bits of offset (max 2^16 blockSize)
	offsetSize, tagSize   uint8  // max 255 (address max 32 bits..., way too much as in fully associative it is 32 bits max)
	source                Datasource
	hitCount, missCount   uint64
	blockSize, dataSize   uint16 // max 65_535 byte for a single data (same as block size)
	maxWays               uint16 // max 65_535, number of maximum ways
	repol                 RePol
}

// CreateCache create a new cache regarding the options given
func CreateCache(sets, blockSize, dataSize, ways uint16, source Datasource, pol RePol) (*Cache, error) {
	// Simple verification for the parameters
	if dataSize > blockSize || dataSize == 0 || blockSize%dataSize != 0 {
		return nil, errors.New("CACHE: The given data are not good")
	}

	// open the source
	err := source.Open()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("CACHE: Cannot open the datasource: %s", err))
	}

	// Calculate the different size
	indexSize := uint8(math.Log2(float64(sets)))
	offsetSize := uint8(math.Log2(float64(blockSize)))
	tagSize := 32 - indexSize - offsetSize

	c := &Cache{
		sets:       make([]*set, sets),
		indexMask:  CalculateMask(indexSize),
		offsetMask: CalculateMask(offsetSize),
		offsetSize: offsetSize,
		tagSize:    tagSize,
		source:     source,
		hitCount:   0,
		missCount:  0,
		blockSize:  blockSize,
		dataSize:   dataSize,
		maxWays:    ways,
		repol:      pol,
	}

	// Create the sets
	for i := uint16(0); i < sets; i++ {
		c.sets[i] = createSet(c)
	}
	return c, nil
}

// Get data at this address using the cache
func (c *Cache) Get(address uint32) []byte {
	// get last 9 bits for index
	index := (address >> c.offsetSize) & c.indexMask
	return c.sets[index].get(address)
}

// ResetCounters resets the hits and misses counters
func (c *Cache) ResetCounters() {
	c.hitCount = 0
	c.missCount = 0
}

// Close closes the cache and the datasource
func (c *Cache) Close() error {
	err := c.source.Close()
	if err != nil {
		return errors.New(fmt.Sprintf("CACHE: Cannot close the source: %s", err))
	}
	return nil
}

// GetCounters gives counter of hits and misses of the cache
func (c *Cache) GetCounters() (hits, misses uint64) {
	return c.hitCount, c.missCount
}

// CalculateMask generates a mask (1 at the LSB)
func CalculateMask(size uint8) uint32 {
	res := uint32(0b0)
	for i := uint8(0); i < size; i++ {
		if res > 0 {
			res = res<<1 + 0b1
		} else {
			res = 0b1
		}
	}
	return res
}
