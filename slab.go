package main

// Slab memory allocation

// Initialise the slab as a channel of blocks, allocating them as required and
// pushing them back on the slab. This reduces garbage collection.
type slab chan []byte

func newSlab(blockSize int, numBlocks int) slab {
	s := make(slab, numBlocks)
	for i := 0; i < numBlocks; i++ {
		s <- make([]byte, blockSize)
	}
	return s
}

func (s slab) Alloc() (x []byte) {
	return <-s
}

func (s slab) Free(x []byte) {
	// Check we are using the right dimensions
	x = x[:cap(x)]
	s <- x
}
