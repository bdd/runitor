package internal

func Min[T ~int | ~uint](x, y T) T {
	if x < y {
		return x
	}

	return y
}

func Max[T ~int | ~uint](x, y T) T {
	if x > y {
		return x
	}

	return y
}
