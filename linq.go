package linq

import (
	"errors"
	"sort"
)

type Queryable struct {
	values []interface{}
	err    error
	less   func(this, that interface{}) bool
}

func (q Queryable) Len() int           { return len(q.values) }
func (q Queryable) Swap(i, j int)      { q.values[i], q.values[j] = q.values[j], q.values[i] }
func (q Queryable) Less(i, j int) bool { return q.less(q.values[i], q.values[j]) }

var (
	ErrNilFunc         = errors.New("linq: passed evaluation function is nil")
	ErrNilInput        = errors.New("linq: nil input passed to From")
	ErrNoElement       = errors.New("linq: element satisfying the conditions does not exist")
	ErrNegativeParam   = errors.New("linq: parameter cannot be negative")
	ErrUnsupportedType = errors.New("linq: sorting this type with Order is not supported, use OrderBy")
)

func From(input []interface{}) Queryable {
	var _err error
	if input == nil {
		_err = ErrNilInput
	}
	return Queryable{
		values: input,
		err:    _err}
}

func (q Queryable) Results() ([]interface{}, error) {
	return q.values, q.err
}

func (q Queryable) Where(f func(interface{}) (bool, error)) (r Queryable) {
	if q.err != nil {
		r.err = q.err
		return r
	}
	if f == nil {
		r.err = ErrNilFunc
		return
	}

	for _, i := range q.values {
		ok, err := f(i)
		if err != nil {
			r.err = err // TODO add extra messages
			return r
		}
		if ok {
			r.values = append(r.values, i)
		}
	}
	return r
}

func (q Queryable) Select(f func(interface{}) (interface{}, error)) (r Queryable) {
	if q.err != nil {
		r.err = q.err
		return r
	}
	if f == nil {
		r.err = ErrNilFunc
		return
	}

	for _, i := range q.values {
		val, err := f(i)
		if err != nil {
			r.err = err // TODO add extra messages
			return r
		}
		r.values = append(r.values, val)
	}
	return
}

func (q Queryable) Distinct() (r Queryable) {
	return q.distinct(nil)
}

func (q Queryable) DistinctBy(f func(interface{}, interface{}) (bool, error)) (r Queryable) {
	if f == nil {
		r.err = ErrNilFunc
		return
	}
	return q.distinct(f)
}

func (q Queryable) distinct(f func(interface{}, interface{}) (bool, error)) (r Queryable) {
	if q.err != nil {
		r.err = q.err
		return r
	}

	if f == nil {
		// basic equality comparison using dict
		dict := make(map[interface{}]bool)
		for _, v := range q.values {
			if _, ok := dict[v]; !ok {
				dict[v] = true
			}
		}
		res := make([]interface{}, len(dict))
		i := 0
		for key, _ := range dict {
			res[i] = key
			i++
		}
		r.values = res
	} else {
		// use equality comparer and bool flags for each item
		// here we check all a[i]==a[j] i<j, practically worst case
		// for this is O(N^2) where all elements are different and best case
		// is O(N) where all elements are the same
		// pick lefthand side value of the comparison in the result
		l := len(q.values)
		results := make([]interface{}, 0)
		included := make([]bool, l)
		for i := 0; i < l; i++ {
			if included[i] {
				continue
			}
			for j := i + 1; j < l; j++ {
				equals, err := f(q.values[i], q.values[j])
				if err != nil {
					r.err = err
					return
				}
				if equals {
					included[j] = true // don't include righthand side value
				}
			}
			results = append(results, q.values[i])
		}
		r.values = results
	}
	return
}

func (q Queryable) Union(in []interface{}) (r Queryable) {
	if q.err != nil {
		r.err = q.err
		return
	}
	var set map[interface{}]bool = make(map[interface{}]bool)

	for _, v := range q.values {
		if _, ok := set[v]; !ok {
			set[v] = true
		}
	}
	for _, v := range in {
		if _, ok := set[v]; !ok {
			set[v] = true
		}
	}
	r.values = make([]interface{}, len(set))
	i := 0
	for k, _ := range set {
		r.values[i] = k
		i++
	}
	return
}

func (q Queryable) Intersect(in []interface{}) (r Queryable) {
	if q.err != nil {
		r.err = q.err
		return
	}
	var set map[interface{}]bool = make(map[interface{}]bool)
	var intersection map[interface{}]bool = make(map[interface{}]bool)

	for _, v := range q.values {
		if _, ok := set[v]; !ok {
			set[v] = true
		}
	}
	for _, v := range in {
		if _, ok := set[v]; ok {
			delete(set, v)
			if _, added := intersection[v]; !added {
				intersection[v] = true
			}
		}
	}
	r.values = make([]interface{}, len(intersection))
	i := 0
	for k, _ := range intersection {
		r.values[i] = k
		i++
	}
	return
}

func (q Queryable) Except(except []interface{}) (r Queryable) {
	if q.err != nil {
		r.err = q.err
		return
	}
	var set map[interface{}]bool = make(map[interface{}]bool)

	for _, v := range q.values {
		if _, ok := set[v]; !ok {
			set[v] = true
		}
	}
	for _, v := range except {
		delete(set, v)
	}
	r.values = make([]interface{}, len(set))
	i := 0
	for k, _ := range set {
		r.values[i] = k
		i++
	}
	return
}

func (q Queryable) Count() (count int, err error) {
	return len(q.values), q.err
}

func (q Queryable) CountBy(f func(interface{}) (bool, error)) (c int, err error) {
	if q.err != nil {
		err = q.err
		return
	}
	if f == nil {
		err = ErrNilFunc
		return
	}

	for _, i := range q.values {
		ok, e := f(i)
		if e != nil {
			err = e // TODO add extra messages
			return
		}
		if ok {
			c++
		}
	}
	return
}

func (q Queryable) Any() (exists bool, err error) {
	return len(q.values) > 0, q.err
}

func (q Queryable) AnyWith(f func(interface{}) (bool, error)) (exists bool, err error) {
	if q.err != nil {
		err = q.err
		return
	}
	if f == nil {
		err = ErrNilFunc
		return
	}

	for _, i := range q.values {
		ok, e := f(i)
		if e != nil {
			err = e // TODO add extra messages
			return
		}
		if ok {
			exists = true
			return
		}
	}
	return
}

func (q Queryable) All(f func(interface{}) (bool, error)) (all bool, err error) {
	if q.err != nil {
		err = q.err
		return
	}
	if f == nil {
		err = ErrNilFunc
		return
	}

	all = true // if no elements, result is true
	for _, i := range q.values {
		ok, e := f(i)
		if e != nil {
			err = e // TODO add extra messages
			return
		}
		all = all && ok
	}
	return
}

func (q Queryable) Single(f func(interface{}) (bool, error)) (single bool, err error) {
	if q.err != nil {
		err = q.err
		return
	}
	if f == nil {
		err = ErrNilFunc
		return
	}
	count, e := q.CountBy(f)
	if e != nil {
		err = e
		return
	}
	single = count == 1
	return
}

func (q Queryable) First() (elem interface{}, err error) {
	if q.err != nil {
		err = q.err
		return
	}
	if len(q.values) == 0 {
		err = ErrNoElement
	} else {
		elem = q.values[0]
	}
	return
}

func (q Queryable) FirstOrNil() (elem interface{}, err error) {
	if q.err != nil {
		err = q.err
		return
	}
	if len(q.values) > 0 {
		elem = q.values[0]
	}
	return
}

func (q Queryable) firstBy(f func(interface{}) (bool, error)) (elem interface{}, found bool, err error) {
	if q.err != nil {
		err = q.err
		return
	}
	if f == nil {
		err = ErrNilFunc
		return
	}
	for _, i := range q.values {
		ok, e := f(i)
		if e != nil {
			err = e // TODO add extra messages
			return
		}
		if ok {
			elem = i
			found = true
			break
		}
	}
	return
}

func (q Queryable) FirstBy(f func(interface{}) (bool, error)) (elem interface{}, err error) {
	var found bool
	elem, found, err = q.firstBy(f)

	if err == nil && !found {
		err = ErrNoElement
	}
	return
}

func (q Queryable) FirstOrNilBy(f func(interface{}) (bool, error)) (elem interface{}, err error) {
	elem, found, err := q.firstBy(f)
	if !found {
		elem = nil
	}
	return
}

func (q Queryable) Last() (elem interface{}, err error) {
	if q.err != nil {
		err = q.err
		return
	}
	if len(q.values) == 0 {
		err = ErrNoElement
	} else {
		elem = q.values[len(q.values)-1]
	}
	return
}

func (q Queryable) LastOrNil() (elem interface{}, err error) {
	if q.err != nil {
		err = q.err
		return
	}
	if len(q.values) > 0 {
		elem = q.values[len(q.values)-1]
	}
	return
}

func (q Queryable) lastBy(f func(interface{}) (bool, error)) (elem interface{}, found bool, err error) {
	if q.err != nil {
		err = q.err
		return
	}
	if f == nil {
		err = ErrNilFunc
		return
	}
	for i := len(q.values) - 1; i >= 0; i-- {
		item := q.values[i]
		ok, e := f(item)
		if e != nil {
			err = e // TODO add extra messages
			return
		}
		if ok {
			elem = item
			found = true
			break
		}
	}
	return
}

func (q Queryable) LastBy(f func(interface{}) (bool, error)) (elem interface{}, err error) {
	var found bool
	elem, found, err = q.lastBy(f)

	if err == nil && !found {
		err = ErrNoElement
	}
	return
}

func (q Queryable) LastOrNilBy(f func(interface{}) (bool, error)) (elem interface{}, err error) {
	elem, found, err := q.lastBy(f)
	if !found {
		elem = nil
	}
	return
}

func (q Queryable) Reverse() (r Queryable) {
	if q.err != nil {
		r.err = q.err
		return
	}
	c := len(q.values)
	j := 0
	r.values = make([]interface{}, c)
	for i := c - 1; i >= 0; i-- {
		r.values[j] = q.values[i]
		j++
	}
	return
}

func (q Queryable) Take(n int) (r Queryable) {
	if q.err != nil {
		r.err = q.err
		return
	}
	if n < 0 {
		n = 0
	}
	if n >= len(q.values) {
		n = len(q.values)
	}
	r.values = q.values[:n]
	return
}

func (q Queryable) Skip(n int) (r Queryable) {
	if q.err != nil {
		r.err = q.err
		return
	}
	if n < 0 {
		n = 0
	}
	if n >= len(q.values) {
		n = len(q.values)
	}
	r.values = q.values[n:]
	return
}

//TODO document: only sorts int, string, float64
func (q Queryable) Order() (r Queryable) {
	if q.err != nil {
		r.err = q.err
		return
	}

	if len(q.values) > 0 {
		f := q.values[0]
		if _, ints := f.(int); ints {
			vals := toInts(q.values)
			sort.Ints(vals)
			r.values = intsToInterface(vals)
		} else if _, strings := f.(string); strings {
			vals := toStrings(q.values)
			sort.Strings(vals)
			r.values = stringsToInterface(vals)
		} else if _, float64s := f.(float64); float64s {
			vals := toFloat64s(q.values)
			sort.Float64s(vals)
			r.values = float64sToInterface(vals)
		} else {
			r.err = ErrUnsupportedType
		}
	}
	return
}

func (q Queryable) OrderBy(less func(this interface{}, that interface{}) bool) (r Queryable) {
	if q.err != nil {
		r.err = q.err
		return
	}
	if less == nil {
		r.err = ErrNilFunc
		return
	}
	r.less = less
	r.values = make([]interface{}, len(q.values))
	_ = copy(r.values, q.values)
	sort.Sort(r)
	return
}