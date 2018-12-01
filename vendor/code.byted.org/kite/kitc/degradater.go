package kitc

import (
	"strconv"
)

// Degradater used by DegradationMW
type Degradater struct {
	storage KVStorage
}

// NewDegradater .
func NewDegradater(storage KVStorage) *Degradater {
	return &Degradater{storage: storage}
}

// GetDegradationPercent gets value from ETCD
func (d *Degradater) GetDegradationPercent(key string) (int, error) {
	val, err := d.storage.GetOrSet(key, "0")
	if err != nil {
		return 0, err
	}

	per, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return per, nil
}

// RandomPercent gets a random number between [0, 100]
func (d *Degradater) RandomPercent() int {
	return defaultSafeRander.Intn(101)
}
