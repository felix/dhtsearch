package dhtsearch

type tagStore interface {
	saveTag(string) (int, error)
}
