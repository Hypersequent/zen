// Package testmodels holds the zen tests' types that cannot be declared inside
// a test function: generic types, mutually recursive types, and the types they
// use. Being in another package, they also serve to test name clashes with the
// tests' local types.
package testmodels

type GenericModel struct {
	ID string
}

type GenericPair[T any, U any] struct {
	First  T
	Second U
}

type StringIntPair GenericPair[string, int]

type PairMap[K comparable, T any, U any] struct {
	Items map[K]GenericPair[T, U] `json:"items"`
}

type EmbeddedIntPair struct {
	GenericPair[int, int]
}

type EmbeddedIntTriplet struct {
	GenericPair[int, GenericPair[int, int]]
}

type EmbeddedIntModelPair struct {
	GenericPair[int, GenericModel]
}

// Wrapper mimics a generic optional type like 4d63.com/optional.Optional[T].
type Wrapper[T any] struct{ Value T }

type CyclicA struct {
	B *CyclicB
}

type CyclicB struct {
	A *CyclicA
}

type User struct {
	Email string
}

type Node struct {
	Children []Node
}
