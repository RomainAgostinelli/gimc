package gimc

// fifo is the struct use to represent and manage a fifo replacement policy
type fifo struct {
	order []uint32 // fifo for the incoming tag, used for replacement
}

// Implement the replacement algorithm and return the tag to remove
// The returned element is directly update regarding the algorithm
func (f *fifo) toReplace() uint32 {
	first:= f.order[0]
	f.order = f.order[1:] // remove the first
	return first
}

// Must be called when hit, part of the replacement algorithm
func (f *fifo) hit(tag uint32) {
 // nothing to do for FIFO
}

// Must be called when miss, part of the replacement algorithm
func (f *fifo) miss(tag uint32) {
	// Add new entry at the end
	f.order = append(f.order, tag)
}
