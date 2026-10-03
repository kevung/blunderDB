package storage

import "testing"

func TestClubRanking_OrdersAcrossTenantsAndSharesRanks(t *testing.T) {
	tables := []Tagged[[]PlayerRow]{
		{Tenant: "2", Item: []PlayerRow{
			{Name: "Paul", Decisions: 100, PR: 8},
			{Name: "Rival", Decisions: 0},
		}},
		{Tenant: "3", Item: []PlayerRow{
			{Name: "Paul", Decisions: 50, PR: 8},
			{Name: "Anne", Decisions: 80, PR: 4.5},
			{Name: "Noise", Decisions: 3, PR: 1},
		}},
	}
	got := ClubRanking(tables, nil, 10)
	want := []struct {
		tenant, name string
		rank         int
	}{
		{"3", "Anne", 1},
		{"2", "Paul", 2},
		{"3", "Paul", 2},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d rows, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Tenant != w.tenant || got[i].Player.Name != w.name || got[i].Rank != w.rank {
			t.Errorf("row %d = (%s, %s, %d), want (%s, %s, %d)", i, got[i].Tenant, got[i].Player.Name, got[i].Rank, w.tenant, w.name, w.rank)
		}
	}
}

func TestClubRanking_UnmeasuredRowsAreUnranked(t *testing.T) {
	got := ClubRanking([]Tagged[[]PlayerRow]{{Tenant: "1", Item: []PlayerRow{
		{Name: "Zero"}, {Name: "Some", Decisions: 5, PR: 12},
	}}}, nil, 0)
	if len(got) != 2 || got[0].Player.Name != "Some" || got[0].Rank != 1 || got[1].Rank != 0 {
		t.Fatalf("got %+v, want Some ranked 1 then Zero unranked", got)
	}
}

func TestClubRanking_OnlyKeepsTheListedPlayersOfTheirTenant(t *testing.T) {
	tables := []Tagged[[]PlayerRow]{
		{Tenant: "2", Item: []PlayerRow{{Name: "Paul", Decisions: 10, PR: 5}, {Name: "Rival", Decisions: 10, PR: 3}}},
		{Tenant: "3", Item: []PlayerRow{{Name: "Paul", Decisions: 10, PR: 4}}},
	}
	got := ClubRanking(tables, []ClubPlayer{{Tenant: "2", Name: "  paul "}}, 0)
	if len(got) != 1 || got[0].Tenant != "2" || got[0].Player.Name != "Paul" {
		t.Fatalf("got %+v, want only tenant 2's Paul", got)
	}
}
