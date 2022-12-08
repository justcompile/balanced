package types

type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(value ...T) {
	for _, v := range value {
		s[v] = struct{}{}
	}
}

// Diff returns a new set with elements in the set that are in `s` but not `b`.
func (s Set[T]) Diff(b Set[T]) Set[T] {
	ret := make(Set[T])

	for k := range s {
		if !b.Has(k) {
			ret.Add(k)
		}
	}

	return ret
}

// Intersect returns a new set with elements in the set that are in `s` AND `b`.
func (s Set[T]) Intersect(b Set[T]) Set[T] {
	ret := make(Set[T])

	for k := range s {
		if b.Has(k) {
			ret.Add(k)
		}
	}

	return ret
}

func (s Set[T]) Remove(v T) {
	delete(s, v)
}

func (s Set[T]) Has(animal T) bool {
	_, ok := s[animal]
	return ok
}
