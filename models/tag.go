package models

type tagStore interface {
	saveTag(string) (int, error)
}
