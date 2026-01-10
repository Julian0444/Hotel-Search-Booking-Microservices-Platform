package hotels

import (
	"search-api/internal/dao/hotels"
)

type Mock struct {
	data map[string]hotels.Hotel
}

func NewMock() Mock {
	return Mock{
		data: make(map[string]hotels.Hotel),
	}
}
