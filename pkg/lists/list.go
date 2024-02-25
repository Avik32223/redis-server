package lists

type Element struct {
	Value      any
	next, prev *Element
	list       *List
}

type List struct {
	root *Element
	len  int
}

func NewList() *List {
	l := new(List)
	root := new(Element)
	l.root = root
	l.root.next = root
	l.root.prev = root
	l.len = 0
	return l
}

func (l *List) Len() int { return l.len }

func (l *List) insert(e, at *Element) *Element {
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	e.list = l
	l.len++
	return e
}

func (l *List) Prepend(v any) *Element {
	e := &Element{
		Value: v,
	}
	return l.insert(e, l.root)
}

func (l *List) Append(v any) *Element {
	e := &Element{
		Value: v,
	}
	return l.insert(e, l.root.prev)
}

func (l *List) ToSlice() []any {
	res := make([]any, 0)
	root := l.root
	head := root.next
	for head != root {
		res = append(res, head.Value)
		head = head.next
	}
	return res
}
