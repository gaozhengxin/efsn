package main

import (
	"fmt"
	"math/big"
	"sort"
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
	return asset, nil
}

func (a *Asset) Align(t uint64) {
	if len(*a) == 0 {
		return
	}
	if !a.IsSorted() {
		a.Sort()
	}
	var s int = -1
	for i, p := range (*a) {
		if p.T >= t {
			s = i - 1
			break
		}
	}
	if s >= 0 {
		*a = (*a)[s:]
		(*a)[0].T = t
	} else {
		p := Point{
			T: t,
			V: big.NewInt(0),
		}
		*a = append(append([]Point{}, p), (*a)...)
	}
	h := 0
	for i := 0; i < len(*a); i++ {
		if (*a)[i].V == nil || (*a)[i].V.Cmp(big.NewInt(0)) == 0 {
			h++
		}
	}
	if h >= 1 {
		*a = (*a)[h-1:]
	}
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
		*c = append(*c, (*a)[i])
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
	if len(*a) <= 1 {
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
	for _, p := range (*a) {
		if p.V.Cmp(big.NewInt(0)) == -1 {
			return false
		}
	}
	return true
}

func (a *Asset) GetAmountByTime(t uint64) *big.Int {
	if (*a)[0].T > t {
		return nil
	}
	var v *big.Int
	for i := 1; i < len(*a); i++ {
		if (*a)[i].T > t {
			return v
		}
		v = (*a)[i].V
	}
	return v
}
