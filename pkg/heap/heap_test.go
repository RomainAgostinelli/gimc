package heap

import (
    "math/rand"
    "testing"
)

func TestHeap(t *testing.T) {
    heap := NewHeap(10)
    for i := uint32(0); i < 10; i++ {
        data := [2]uint32{
            uint32(rand.Intn(15)) + 1, // 1..15
            i,
        }
        err := heap.Add(data)
        if err != nil {
            t.Fatal("Must be capable of adding element")
        }
    }

    min := heap.RemoveMin()
    _ = heap.Add(min)
    min2 := heap.RemoveMin()
    if min[0] != min2[0] || min[1] != min2[1] {
        t.Fatal("Must be the same")
    }
    data := [2]uint32{
        0, // the min added so far
        10, // only one
    }
    heap.Add(data)
    data[0] = 15
    heap.Update(data)
    removeMin := heap.RemoveMin()
    if removeMin[1] == data[1] {
        t.Fatal("Not sifted down when updated")
    }
}