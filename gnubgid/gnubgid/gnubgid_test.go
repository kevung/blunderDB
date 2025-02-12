package gnubgid

import (
	"reflect"
	"testing"
)

func TestPrintHumanReadableMatchKey(t *testing.T) {
	type args struct {
		mk *MatchKey
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// Test cases
		{
			name: "MatchKey",
			args: args{
				mk: &MatchKey{1, 3, 1, 0, 1, 1, 0, 0, [2]int{6, 2}, 3, [2]int{0, 0}},
			},
			wantErr: false,
		},
		{
			name: "MatchKey with cube position error",
			args: args{
				mk: &MatchKey{1, 4, 1, 0, 1, 1, 0, 0, [2]int{6, 2}, 3, [2]int{0, 0}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := PrintHumanReadableMatchKey(tt.args.mk); (err != nil) != tt.wantErr {
				t.Errorf("PrintHumanReadableMatchKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_checkPositionID(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// Test cases
		{
			name: "Good position",
			args: args{
				s: "4HPwATDgc/ABMA",
			},
			want:    "4HPwATDgc/ABMA==",
			wantErr: false,
		},
		{
			name: "Bad position",
			args: args{
				s: "4HPwATDgc/ABM",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkPositionID(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPositionID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkPositionID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateGnubgID(t *testing.T) {
	// data for the tests
	gbgid := GnubgID{
		ID:         "4HPwATDgc/ABMA:cAkAAAAAAAAA",
		PositionID: "4HPwATDgc/ABMA==",
		MatchID:    "cAkAAAAAAAAA",
		Position1:  [25]int{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0},
		Position2:  [25]int{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0},
		MatchKey:   MatchKey{1, 3, 1, 0, 1, 1, 0, 0, [2]int{0, 0}, 0, [2]int{0, 0}},
	}
	gnubgid1 := gbgid
	gnubgid1.ID = "4HPwATDgc/ABMA:cAkAAAAAAAAA"
	gnubgid2 := gbgid
	gnubgid2.ID = "4HPwATDgc/ABMA"
	gnubgid3 := gbgid
	gnubgid3.ID = "cAkAAAAAAAAA"
	gnubgid4 := gbgid
	gnubgid4.ID = ""

	type args struct {
		g *GnubgID
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test1: GNUbgIDis PositionID:MatchID",
			args: args{
				g: &gnubgid1},
			wantErr: false,
		},
		{
			name: "Test2: GNUbgID is PositionID only",
			args: args{
				g: &gnubgid2},
			wantErr: false,
		},
		{
			name: "Test3: GNUbgID is MatchID only",
			args: args{
				g: &gnubgid3},
			wantErr: false,
		},
		{
			name: "Test4: GNUbgID is an empty string",
			args: args{
				g: &gnubgid4},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateGnubgID(tt.args.g); (err != nil) != tt.wantErr {
				t.Errorf("validateGnubgID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getMatchID(t *testing.T) {
	// data for the tests
	gbgid := GnubgID{
		ID:         "4HPwATDgc/ABMA:cAkAAAAAAAAA",
		PositionID: "4HPwATDgc/ABMA==",
		MatchID:    "cAkAAAAAAAAA",
		Position1:  [25]int{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0},
		Position2:  [25]int{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0},
		MatchKey:   MatchKey{1, 3, 1, 0, 1, 1, 0, 0, [2]int{0, 0}, 0, [2]int{0, 0}},
	}
	gnubgid1 := gbgid
	gnubgid1.MatchID = "cAkAAAAAAAAA"
	gnubgid2 := gbgid
	gnubgid2.MatchID = "XXXXXaGVsbG8=" // base64 corrupt

	type args struct {
		g *GnubgID
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// Test cases
		{
			name: "Test1: good MatchID",
			args: args{
				g: &gnubgid1},
			wantErr: false,
		},
		{
			name: "Test2: bad MatchID (corrupt base64)",
			args: args{
				g: &gnubgid2},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := getMatchID(tt.args.g); (err != nil) != tt.wantErr {
				t.Errorf("getMatchID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getPositionID(t *testing.T) {
	// data for the tests
	gbgid := GnubgID{
		ID:         "4HPwATDgc/ABMA:cAkAAAAAAAAA",
		PositionID: "4HPwATDgc/ABMA==",
		MatchID:    "cAkAAAAAAAAA",
		Position1:  [25]int{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0},
		Position2:  [25]int{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0},
		MatchKey:   MatchKey{1, 3, 1, 0, 1, 1, 0, 0, [2]int{0, 0}, 0, [2]int{0, 0}},
	}
	gnubgid1 := gbgid
	gnubgid1.PositionID = "4HPwATDgc/ABMA=="
	gnubgid2 := gbgid
	gnubgid2.PositionID = "XXXXXaGVsbG8=" // base64 corrupt

	type args struct {
		g *GnubgID
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// Test cases
		{
			name: "Test1: good PositionID",
			args: args{
				g: &gnubgid1},
			wantErr: false,
		},
		{
			name: "Test2: bad PositionID (corrupt base64)",
			args: args{
				g: &gnubgid2},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := getPositionID(tt.args.g); (err != nil) != tt.wantErr {
				t.Errorf("getPositionID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadGnubgID(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    GnubgID
		wantErr bool
	}{

		// test cases
		{
			name: "Starting position - money game",
			args: args{
				s: "4HPwATDgc/ABMA:cAkAAAAAAAAA",
			},
			want: GnubgID{
				ID:         "4HPwATDgc/ABMA:cAkAAAAAAAAA",
				PositionID: "4HPwATDgc/ABMA==",
				MatchID:    "cAkAAAAAAAAA",
				Position1:  [25]int{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0},
				Position2:  [25]int{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0},
				MatchKey:   MatchKey{1, 3, 1, 0, 1, 1, 0, 0, [2]int{0, 0}, 0, [2]int{0, 0}},
			},
			wantErr: false,
		},
		{
			name: "PositionID only",
			args: args{
				s: "2+4OAADb7g4AAA",
			},
			want: GnubgID{
				ID:         "2+4OAADb7g4AAA",
				PositionID: "2+4OAADb7g4AAA==",
				MatchID:    "",
				Position1:  [25]int{2, 2, 2, 3, 3, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Position2:  [25]int{2, 2, 2, 3, 3, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				MatchKey:   MatchKey{1, 0, 0, 0, 0, 0, 0, 0, [2]int{0, 0}, 0, [2]int{0, 0}},
			},
			wantErr: false,
		},
		{
			name: "MatchID only",
			args: args{
				s: "cAlrAAAAAAAE",
			},
			want: GnubgID{
				ID:         "cAlrAAAAAAAE",
				PositionID: "",
				MatchID:    "cAlrAAAAAAAE",
				Position1:  [25]int{15, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Position2:  [25]int{15, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				MatchKey:   MatchKey{1, 3, 1, 0, 1, 1, 0, 0, [2]int{6, 2}, 3, [2]int{0, 0}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadGnubgID(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadGnubgID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadGnubgID() = %v, want %v", got, tt.want)
			}
		})
	}
}
