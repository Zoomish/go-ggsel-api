package main

type lineEdit struct {
	op   byte
	text string
}

func lineEdits(a, b []string) []lineEdit {
	n, m := len(a), len(b)
	max := n + m
	v := map[int]int{1: 0}
	trace := make([]map[int]int, 0, max+1)
	for d := 0; d <= max; d++ {
		snap := make(map[int]int, len(v))
		for key, val := range v {
			snap[key] = val
		}
		trace = append(trace, snap)
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1] < v[k+1]) {
				x = v[k+1]
			} else {
				x = v[k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[k] = x
			if x >= n && y >= m {
				return backtrack(trace, a, b)
			}
		}
	}
	return nil
}

func backtrack(trace []map[int]int, a, b []string) []lineEdit {
	x, y := len(a), len(b)
	var rev []lineEdit
	for d := len(trace) - 1; d >= 0; d-- {
		v := trace[d]
		k := x - y
		var prevK int
		if k == -d || (k != d && v[k-1] < v[k+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}
		prevX := v[prevK]
		prevY := prevX - prevK
		for x > prevX && y > prevY {
			x--
			y--
		}
		if d > 0 {
			if x == prevX {
				rev = append(rev, lineEdit{op: '+', text: b[y-1]})
			} else {
				rev = append(rev, lineEdit{op: '-', text: a[x-1]})
			}
		}
		x, y = prevX, prevY
	}
	out := make([]lineEdit, len(rev))
	for i := range rev {
		out[len(rev)-1-i] = rev[i]
	}
	return out
}
