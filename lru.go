package gimc

import (
	"github.com/RomainAgostinelli/gimc/pkg/heap"
	"log"
	"math"
)

// lru is the structure used to implement and represent the Least Recently Used replacement policy.
type lru struct {
	heap    *heap.Heap // Used for LRU replacement algorithm
	lclock  uint32     // logical clock for ordering new entries in the cache
	maxSize uint16     // maxSize is the maximum size of the heap (the maximum number of ways)
}

func (l *lru) toReplace() uint32 {
	min := l.heap.RemoveMin()
	return min[1]
}

func (l *lru) hit(tag uint32) {
	data := [2]uint32{
		l.lclock,
		tag,
	}
	l.heap.Update(data)
	l.lclock++
	l.maybeRebuild()
}

func (l *lru) miss(tag uint32) {
	data := [2]uint32{
		l.lclock,
		tag,
	}
	err := l.heap.Add(data)
	if err != nil {
		log.Fatalln("CACHE SET: Cannot add new entry in the heap when miss happened")
	}
	l.lclock++
	l.maybeRebuild()
}

// Must be called when miss or hit, part of the replacement algorithm
func (l *lru) maybeRebuild() {
	if l.lclock == math.MaxInt32 {
		// need to rebuild the heap for consistency
		l.lclock = 0
		newHeap := heap.NewHeap(int(l.maxSize))
		for i := 0; i < l.heap.Size(); i++ {
			data := l.heap.RemoveMin()
			data[0] = l.lclock
			err := newHeap.Add(data)
			if err != nil {
				log.Fatalln("LRU: Cannot rebuild the heap")
			}
			l.lclock++
		}
	}
}
