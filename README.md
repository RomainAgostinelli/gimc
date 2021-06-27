# Go In-Memory Cache
The goal of this implementation of in-memory cache in go is educative.

It emulates an in-memory, n-way set-associative cache system based on line number on a file.

## For what?
This implementation can be used for accelerating continuous read of a file or data on in external support.

## Example
In the ```cache_test.go``` file, there is an implementation of a setup used for reading
list of hashes that are into a file.