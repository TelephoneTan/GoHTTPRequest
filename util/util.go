package util

func Copy[T any](t T, init ...func(*T)) *T {
	if len(init) > 0 {
		init[0](&t)
	}
	return &t
}

func New[T any](t *T, init ...func(*T)) *T {
	if len(init) > 0 {
		init[0](t)
	}
	return t
}
