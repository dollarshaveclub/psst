package directory

import (
	"context"
	"testing"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

func TestGetMatches(t *testing.T) {
	testGHState := &GH{}
	testGHState.Members = []Member{Member{Login: "test1", Name: "Test 1"}, Member{Login: "test2", Name: ""}}
	testGHState.Info.Teams = []Team{Team{Name: "team1", Members: []string{"test1", "test2"}}, Team{Name: "team2", Members: []string{}}}

	cases := map[string]struct {
		State    *GH
		Lookup   string
		Expected Matches
	}{
		"NoLookupTest": {
			State:  testGHState,
			Lookup: "",
			Expected: Matches{
				Members: []Member{Member{Login: "test1", Name: "Test 1"}, Member{Login: "test2", Name: ""}},
				Teams:   []Team{Team{Name: "team1", Members: []string{"test1", "test2"}}, Team{Name: "team2", Members: []string{}}},
			},
		},
		"StarLookupTest": {
			State:  testGHState,
			Lookup: "*",
			Expected: Matches{
				Members: []Member{Member{Login: "test1", Name: "Test 1"}, Member{Login: "test2", Name: ""}},
				Teams:   []Team{Team{Name: "team1", Members: []string{"test1", "test2"}}, Team{Name: "team2", Members: []string{}}},
			},
		},
		"LookupUserLoginTest": {
			State:  testGHState,
			Lookup: "test1",
			Expected: Matches{
				Members: []Member{Member{Login: "test1", Name: "Test 1"}},
			},
		},
		"LookupUserNameTest": {
			State:  testGHState,
			Lookup: "Test 1",
			Expected: Matches{
				Members: []Member{Member{Login: "test1", Name: "Test 1"}},
			},
		},
		"LookupUserPartialTest": {
			State:  testGHState,
			Lookup: "test",
			Expected: Matches{
				Members: []Member{Member{Login: "test1", Name: "Test 1"}, Member{Login: "test2", Name: ""}},
			},
		},
		"LookupTeamNameTest": {
			State:  testGHState,
			Lookup: "team1",
			Expected: Matches{
				Teams: []Team{Team{Name: "team1", Members: []string{"test1", "test2"}}},
			},
		},
		"LookupTeamPartialTest": {
			State:  testGHState,
			Lookup: "team",
			Expected: Matches{
				Teams: []Team{Team{Name: "team1", Members: []string{"test1", "test2"}}, Team{Name: "team2", Members: []string{}}},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := c.State.GetMatches(c.Lookup)
			t.Logf("Name: %s, got: %+v, expected: %+v", name, got, c.Expected)
			if !checkMembers(got.Members, c.Expected.Members) {
				t.Errorf("Name: %s members, got: %+v, expected %+v", name, got, c.Expected)
			}
			if !checkTeams(got.Teams, c.Expected.Teams) {
				t.Errorf("Name: %s teams, got: %+v, expected %+v", name, got, c.Expected)
			}
		})
	}
}

func TestIsMember(t *testing.T) {
	testGHState := &GH{}
	testGHState.Members = []Member{Member{Login: "test1", Name: "Test 1"}, Member{Login: "test2", Name: ""}}
	testGHState.Info.Teams = []Team{Team{Name: "team1", Members: []string{"test1", "test2"}}, Team{Name: "team2", Members: []string{}}}

	cases := map[string]struct {
		State    *GH
		Lookup   string
		Expected bool
	}{
		"TestMemberExists": {
			State:    testGHState,
			Lookup:   "test1",
			Expected: true,
		},
		"TestMemberMissing": {
			State:    testGHState,
			Lookup:   "notthere",
			Expected: false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, got := c.State.IsMember(c.Lookup)
			if got != c.Expected {
				t.Errorf("Name: %s, got: %v, expected: %v", name, got, c.Expected)
			}
		})
	}
}

func TestIsTeam(t *testing.T) {
	testGHState := &GH{}
	testGHState.Members = []Member{Member{Login: "test1", Name: "Test 1"}, Member{Login: "test2", Name: ""}}
	testGHState.Info.Teams = []Team{Team{Name: "team1", Members: []string{"test1", "test2"}}, Team{Name: "team2", Members: []string{}}}

	cases := map[string]struct {
		State    *GH
		Lookup   string
		Expected bool
	}{
		"TestTeamExists": {
			State:    testGHState,
			Lookup:   "team1",
			Expected: true,
		},
		"TestTeamMissing": {
			State:    testGHState,
			Lookup:   "notthere",
			Expected: false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, got := c.State.IsTeam(c.Lookup)
			if got != c.Expected {
				t.Errorf("Name: %s, got: %v, expected: %v", name, got, c.Expected)
			}
		})
	}
}

func TestGetTeamMembers(t *testing.T) {
	testGHState := &GH{}
	testGHState.Members = []Member{Member{Login: "test1", Name: "Test 1"}, Member{Login: "test2", Name: ""}}
	testGHState.Info.Teams = []Team{Team{Name: "team1", Members: []string{"test1", "test2"}}, Team{Name: "team2", Members: []string{}}}

	cases := map[string]struct {
		State    *GH
		Lookup   string
		Expected []string
	}{
		"TestTeamExists": {
			State:    testGHState,
			Lookup:   "team1",
			Expected: []string{"test1", "test2"},
		},
		"TestTeamMissing": {
			State:    testGHState,
			Lookup:   "notthere",
			Expected: []string{},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := c.State.GetTeamMembers(c.Lookup)
			if len(got) != len(c.Expected) {
				t.Errorf("Name: %s, got: %d, expected: %d", name, len(got), len(c.Expected))
			}
			for i := range got {
				if got[i] != c.Expected[i] {
					t.Errorf("Name: %s, got: %v, expected: %v", name, got, c.Expected)
				}
			}
		})
	}
}

type UsersServiceTester struct {
	Login string
	Err   error
}

func (u UsersServiceTester) Get(ctx context.Context, name string) (*github.User, *github.Response, error) {
	return &github.User{Login: &u.Login}, nil, u.Err
}

func TestWhoami(t *testing.T) {
	us := UsersServiceTester{Login: "test1"}

	testGHState := &GH{
		UsersService: us,
	}

	type expected struct {
		login string
		err   error
	}

	cases := map[string]struct {
		State    *GH
		Lookup   string
		Expected expected
	}{
		"TestWhoami": {
			State:  testGHState,
			Lookup: "",
			Expected: expected{
				login: "test1",
				err:   nil,
			},
		},
		"TestWhoamiError": {
			State: &GH{
				UsersService: UsersServiceTester{Login: "", Err: errors.New("bad username")},
			},
			Lookup: "",
			Expected: expected{
				login: "",
				err:   errors.New("unable to get authenticated user's login: bad username"),
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			login, err := c.State.Whoami()
			if login != c.Expected.login {
				t.Errorf("Name: %s, got: %s, expected: %s", name, login, c.Expected.login)
			}
			if err != nil && c.Expected.err != nil {
				if err.Error() != c.Expected.err.Error() {
					t.Errorf("Name: %s, got: %v, expected: %v", name, err, c.Expected.err)
				}
			}
		})
	}
}

func checkMembers(g []Member, e []Member) bool {
	if len(g) != len(e) {
		return false
	}

	for _, s := range g {
		found := false
		for _, q := range e {
			if s.Login == q.Login && s.Name == q.Name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	for _, s := range e {
		found := false
		for _, q := range g {
			if s.Login == q.Login && s.Name == q.Name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func checkTeams(g []Team, e []Team) bool {
	if len(g) != len(e) {
		return false
	}

	for _, s := range g {
		found := false
		for _, q := range e {
			if s.Name == q.Name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	for _, s := range e {
		found := false
		for _, q := range g {
			if s.Name == q.Name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
