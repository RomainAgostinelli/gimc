package heap

import (
    "errors"
    "fmt"
)

type Heap struct {
    harr        [][2]uint32
    maxSize     int
    currentSize int
}

// NewHeap create a new heap with a maximum size given in parameter
func NewHeap(size int) *Heap {
    return &Heap{maxSize: size, currentSize: 0}
}

// percolateUp push up the element at position i by swapping until it is at the right position
func (h *Heap) percolateUp(i int) {
    for i > 0 && h.harr[parent(i)][0] > h.harr[i][0] {
        h.harr[i], h.harr[parent(i)] = h.harr[parent(i)], h.harr[i]
        i = parent(i)
    }
}

// siftDown push down the element at position i by swapping until  it is at the right position
func (h *Heap) siftDown(i int) {
    l := leftChild(i)
    r := rightChild(i)
    smallest := i
    if r < len(h.harr) && h.harr[r][0] < h.harr[i][0] {
        smallest = r
    }
    if l < len(h.harr) && h.harr[l][0] < h.harr[smallest][0] {
        smallest = l
    }
    if smallest != i {
        h.harr[i], h.harr[smallest] = h.harr[smallest], h.harr[i]
        h.siftDown(smallest)
    }
}

func (h *Heap) Size() int {
    return h.currentSize
}

// Add an element in the heap. The priority (key used for the heap) is the first element of the parameter "val"
func (h *Heap) Add(val [2]uint32) error {
    if h.currentSize == h.maxSize {
        return errors.New(fmt.Sprintf("HEAP: Max size reached (%d/%d)", h.currentSize, h.maxSize))
    }
    h.harr = append(h.harr, val)
    h.percolateUp(len(h.harr) - 1)
    h.currentSize++
    return nil
}

// RemoveMin removes the min element regarding its key (first element of the arrays present in heap)
func (h *Heap) RemoveMin() [2]uint32 {
    if h.currentSize == 0 {
        return [2]uint32{}
    }
    // store the root (minimum)
    min := h.harr[0]
    // put the max to the top and sift down
    h.harr[0] = h.harr[len(h.harr) - 1]
    h.harr = h.harr[:len(h.harr) - 1] // remove the last element
    h.siftDown(0)
    h.currentSize--
    return min
}

// Update the data given in parameter, assuming the second element of the data is the key and must already be present
// in the heap.
// This is not a treap, so update takes 0(n)
func (h *Heap) Update(data [2]uint32)  {
    var i int
    var val [2]uint32
    for i, val = range h.harr {
        if val[1] == data[1] {
            break
        }
    }
    // now update the new regarding increase / decrease
    if val[0] < data[0] { // increase, sift down
        h.harr[i][0] = data[0]
        h.siftDown(i)
    } else if val[0] > data[0] { // decrease, percolate up
        h.harr[i][0] = data[0]
        h.percolateUp(i)
    }
}

// parent give the parent index of the element at the index i
func parent(i int) int {
    return (i - 1) / 2
}

// leftChild gives the left child of the element at index i, may be out of bound if not exists
func leftChild(i int) int {
    return 2*i + 1
}

// rightChild gives the right child of the element at index i, may be out of bound if not exists
func rightChild(i int) int {
    return 2*i + 2
}
