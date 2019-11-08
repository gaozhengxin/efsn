package main

import (
	"fmt"
	"math/big"
	"sort"
	"time"
)

type Asset []Point

type Point struct {
	T uint64
	V *big.Int
}

func (a *Asset) Sort() {
	sort.Stable(a)
}

func (a *Asset) IsSorted() bool {
	return sort.IsSorted(a)
}

func (a *Asset) Len() int {
	return len(*a)
}

func (a *Asset) Less(i, j int) bool {
	return (*a)[i].T < (*a)[j].T
}

func (a *Asset) Swap(i, j int) {
	tmp := (*a)[i]
	(*a)[i] = (*a)[j]
	(*a)[j] = tmp
}

// end = 0 : infinite
func NewAsset(amount *big.Int, start uint64, end uint64) (*Asset, error) {
	if end != 0 && end <= start {
		return nil, fmt.Errorf("end time must be greater than start time")
	}
	asset := new(Asset)
	*asset = append(*asset, Point{T:start, V:amount})
	if end != 0 {
		*asset = append(*asset, Point{T:end, V:big.NewInt(0)})
	}
	asset.Reduce()
	return asset, nil
}

func (a *Asset) Align(t uint64) {
	if len(*a) == 0 {
		return
	}
	a.Reduce()

	var k int = -1
	var p Point = Point{
		T:t,
		V:big.NewInt(0),
	}

	for i := 0; i < len(*a); i++ {
		if (*a)[i].T > t {
			k = i
			v := big.NewInt(0)
			if i > 0 {
				v = (*a)[i-1].V
			}
			p = Point{
				T:t,
				V:v,
			}
			break
		}
	}
	ps := []Point{}
	if k >= 0 {
		ps = append([]Point{}, p)
		ps = append(ps, (*a)[k:]...)
	} else {
		ps = append([]Point{}, Point{T:t, V:(*a)[len(*a) - 1].V})
	}
	*a = ps
	a.Reduce()
}

func (a *Asset) Reduce() {
	if len(*a) == 0 {
		return
	}
	if !a.IsSorted() {
		a.Sort()
	}
	c := new(Asset)
	*c = append(*c, (*a)[0])
	for i := 1; i < len(*a); i++ {
		if (*a)[i-1].V.Cmp((*a)[i].V) == 0 {
			continue
		}
		if (*a)[i-1].T == (*a)[i].T {
			(*c)[len(*c) - 1].V = new(big.Int).Add((*c)[len(*c) - 1].V, (*a)[i].V)
			continue
		}
		*c = append(*c, (*a)[i])
	}
	if len(*c) > 1 && (*c)[len(*c) - 1].T > uint64(9223372036854775807) {
		*c = (*c)[:len(*c) - 1]
	}
	*a = *c
}

func (a *Asset) ReduceDiff() {
	d := new(Asset)
	*d = append(*d, (*a)[0])
	j := 1
	for i := 1; i < len(*a); i++ {
		if (*a)[i].T == (*a)[i-1].T {
			(*d)[j-1].V = new(big.Int).Add((*d)[j-1].V, (*a)[i].V)
			continue
		}
		*d = append(*d, (*a)[i])
		j++
	}
	*a = *d
}

func (a *Asset) Diff() (*Asset) {
	d := new(Asset)
	if len(*a) < 1 {
		return nil
	} else {
		(*d) = append([]Point{}, (*a)[0])
		for i := 1; i < len(*a); i++ {
			p := Point{
				T: (*a)[i].T,
				V: new(big.Int).Sub((*a)[i].V, (*a)[i-1].V),
			}
			*d = append((*d), p)
		}
	}
	return d
}

func (a *Asset) Cumulate() (*Asset) {
	if !a.IsSorted() {
		a.Sort()
	}
	c := new(Asset)
	*c = append(*c,  (*a)[0])
	for i := 1; i < len(*a); i++ {
		p := Point{
			T: (*a)[i].T,
			V: new(big.Int).Add((*c)[i-1].V, (*a)[i].V),
		}
		*c = append((*c), p)
	}
	return c
}

func (a *Asset) Add(b *Asset) (*Asset) {
	if len(*a) == 0 {
		return b
	}
	if len(*b) == 0 {
		return a
	}
	da := a.Diff()
	db := b.Diff()
	d := append((*da), (*db)...)
	d.Sort()
	d.ReduceDiff()
	return d.Cumulate()
}

func (a *Asset) IsNonneg() bool {
	b := Copy(a)
	b.Align(uint64(time.Now().Unix()))
	for _, p := range (*b) {
		if p.V.Cmp(big.NewInt(0)) == -1 {
			return false
		}
	}
	return true
}

func Inv(a *Asset) (*Asset) {
	b := new(Asset)
	for i := 0; i < len(*a); i++ {
		p := Point{
			T:(*a)[i].T,
			V:new(big.Int).Neg((*a)[i].V),
		}
		(*b) = append(*b, p)
	}
	return b
}

func (a *Asset) Sub(b *Asset) (*Asset) {
	return a.Add(Inv(b))
}

func (a *Asset) Equal(b *Asset) bool {
	today := uint64(GetTodayZero().Unix())
	a.Align(today)
	b.Align(today)
	if len(*a) == len(*b) {
		for i := 0; i < len(*a); i++ {
			if (*a)[i].T != (*b)[i].T || (*a)[i].V.Cmp((*b)[i].V) != 0 {
				return false
			}
		}
		return true
	}
	return false
}

func ZeroAsset() *Asset {
	zero, _ := NewAsset(big.NewInt(0), 0, 0)
	return zero
}

func (a *Asset) GetAmountByTime(t uint64) *big.Int {
	v := big.NewInt(0)
	for i := 0; i < len(*a); i++ {
		if (*a)[i].T > t {
			return v
		}
		v = (*a)[i].V
	}
	return v
}

func Copy(a *Asset) *Asset {
	b := &Asset{}
	*b = make([]Point, len(*a))
	copy(*b, *a)
	return b
}
