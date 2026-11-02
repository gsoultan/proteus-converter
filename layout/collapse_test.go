// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package layout

import "testing"

func TestCollapsedBorderConflicts(t *testing.T) {
	solid1 := cborder{width: 0.75, style: "solid", rank: 4}
	solid2 := cborder{width: 1.5, style: "solid", rank: 1}
	dashed2 := cborder{width: 1.5, style: "dashed", rank: 4}
	hidden := cborder{style: "hidden", rank: 1}
	none := cborder{style: "none", rank: 4}
	cases := []struct {
		name  string
		cands []cborder
		want  cborder
	}{
		{"wider wins over origin", []cborder{solid1, solid2}, solid2},
		{"style breaks width ties", []cborder{dashed2, solid2}, solid2},
		{"origin breaks style ties", []cborder{solid2, {width: 1.5, style: "solid", rank: 3}}, cborder{width: 1.5, style: "solid", rank: 3}},
		{"hidden suppresses everything", []cborder{solid2, hidden, dashed2}, cborder{style: "hidden", rank: 1}},
		{"none loses", []cborder{none, solid1}, solid1},
	}
	for _, c := range cases {
		if got := resolve(c.cands...); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}
