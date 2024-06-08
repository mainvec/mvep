package wou

import (
	"sort"

	"golang.org/x/exp/constraints"
)

// OMap is a map with iteration order
type OMap[K constraints.Ordered, V any] map[K]V

type iterator[K constraints.Ordered, V any] struct {
	i     int
	_keys []K
	_map  map[K]V
}

func (iter *iterator[K, V]) HasNext() bool {
	return int(iter.i) < len(iter._keys)
}

func (iter *iterator[K, V]) Next() (K, V) {
	if !iter.HasNext() {
		panic("iteration has no more elements, shold us HasNext")
	}
	k := iter._keys[iter.i]
	iter.i++
	return k, iter._map[k]

}

// func IteratorByKey[K constraints.Ordered, V any](_omap interface{}) iterator[K, V] {
// 	omap, ok := _omap.(OMap[K, V])
// 	if !ok {
// 		panic("iterationByKey needs OMap")
// 	}
// 	return iteratorByKey(omap)
// }

// Create a iterator for the map, ordered by map keys
func IteratorByKey[K constraints.Ordered, V any](omap map[K]V) iterator[K, V] {

	keys := make([]K, 0, len(omap))
	for k := range omap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return iterator[K, V]{
		i:     0,
		_keys: keys,
		_map:  omap,
	}
}

func (omap OMap[K, V]) IterateByKey() iterator[K, V] {
	return IteratorByKey(omap)
}

type mapItem[K constraints.Ordered, V any] struct {
	_k K
	_v V
}

// Create a iterator for the map, ordered by map vlaues, using the lessFunc
func IterateByValue[K constraints.Ordered, V any](omap map[K]V, lessFunc func(i, j V) bool) iterator[K, V] {

	items := make([]*mapItem[K, V], 0, len(omap))
	for k, v := range omap {
		items = append(items, &mapItem[K, V]{
			_k: k,
			_v: v,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return lessFunc(items[i]._v, items[j]._v)
	})

	keys := make([]K, 0, len(omap))
	for _, item := range items {
		keys = append(keys, item._k)
	}
	return iterator[K, V]{
		i:     0,
		_keys: keys,
		_map:  omap,
	}
}

// Create a iterator for the map, ordered by map vlaues, using the lessFunc
func (omap OMap[K, V]) IterateByValue(lessFunc func(i, j V) bool) iterator[K, V] {
	//Can only be used by direct OMap. Cannot be used by structs "inherting" OMap
	return IterateByValue(omap, lessFunc)
}
