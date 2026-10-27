package main

import "container/list"

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}

type LRUCache[K comparable, V any] struct {
	capacity int
	ll       list.List
	items    map[K]*list.Element
}

type entry[K comparable, V any] struct {
	key   K
	value V
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{capacity: capacity, ll: list.List{}, items: make(map[K]*list.Element, capacity)}
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {

	if c.capacity <= 0 {
		return
	}

	elem, OK := c.items[key]

	if !OK {
		return
	}

	c.ll.MoveToFront(elem)
	e := elem.Value

	if ent, ok := e.(entry[K, V]); ok {
		return ent.value, true
	}

	return

}


func (c *LRUCache[K, V]) Set(key K, value V) {

	if c.capacity <= 0 {
		return
	}

	elem, ok := c.items[key]

	if ok {
		elem.Value = entry[K, V]{key: key, value: value}
		return
	}

	if len(c.items) + 1 > c.capacity {
		last := c.ll.Back()
		ent := last.Value.(entry[K, V])
		delete(c.items, ent.key)
		c.ll.Remove(last)
	}
	
	newElem := c.ll.PushFront(entry[K, V]{key: key, value: value})
	c.items[key] = newElem


}

