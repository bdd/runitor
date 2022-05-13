package internal

type Optional[T any] struct {
	defined bool
	value   T
}

func (o Optional[T]) IsDefined() bool {
	return o.defined
}

func (o Optional[T]) Get() (T, bool) {
	return o.value, o.defined
}

func Some[T any](val T) Optional[T] {
	return Optional[T]{defined: true, value: val}
}

func None[T any]() Optional[T] {
	return Optional[T]{}
}
