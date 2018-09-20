package directory

import (
	"os"
	"sort"
)

const (
	cacheTTL = 60.0 // 60 minute TTL
)

var cacheDir = os.ExpandEnv("${HOME}/.psst/cache")

// Backend allows us to have an easy way to get information from GitHub for members and teams
type Backend interface {
	GetMatches(string) Matches
	GetMembers() []Member
	GetTeams() []Team
	GetTeamMembers(string) []string
	GetActiveMemberTeams() []string
	IsMember(string) (string, bool)
	IsTeam(string) (string, bool)
	Whoami() (string, error)
}

// Info is the basic information required by all directory implementations
type Info struct {
	Org               string
	ActiveMemberTeams []string
	Members           []Member
	Teams             []Team
}

// Member contains basic info about a member
type Member struct {
	Login string
	Name  string
}

// Team contains basic info about Team or group
type Team struct {
	Name    string
	Members []string
}

// Matches allows us to return both usernames and team names as single type
type Matches struct {
	Members []Member
	Teams   []Team
}

// ByMembers is the type of a "less" function that defines the ordering of its Member arguments.
type ByMembers func(p1, p2 *Member) bool

var sortMemberLogins = func(m1, m2 *Member) bool {
	return m1.Login < m2.Login
}

// Sort is a method on the function type, By, that sorts the argument slice according to the function.
func (by ByMembers) Sort(members []Member) {
	ms := &memberSorter{
		members: members,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ms)
}

// memberSorter joins a By function and a slice of Members to be sorted.
type memberSorter struct {
	members []Member
	by      func(p1, p2 *Member) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (s *memberSorter) Len() int {
	return len(s.members)
}

// Swap is part of sort.Interface.
func (s *memberSorter) Swap(i, j int) {
	s.members[i], s.members[j] = s.members[j], s.members[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *memberSorter) Less(i, j int) bool {
	return s.by(&s.members[i], &s.members[j])
}

// ByTeams is the type of a "less" function that defines the ordering of its Team arguments.
type ByTeams func(p1, p2 *Team) bool

var sortTeamNames = func(t1, t2 *Team) bool {
	return t1.Name < t2.Name
}

// Sort is a method on the function type, By, that sorts the argument slice according to the function.
func (by ByTeams) Sort(teams []Team) {
	ms := &teamSorter{
		teams: teams,
		by:    by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ms)
}

// teamSorter joins a By function and a slice of Teams to be sorted.
type teamSorter struct {
	teams []Team
	by    func(p1, p2 *Team) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (s *teamSorter) Len() int {
	return len(s.teams)
}

// Swap is part of sort.Interface.
func (s *teamSorter) Swap(i, j int) {
	s.teams[i], s.teams[j] = s.teams[j], s.teams[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *teamSorter) Less(i, j int) bool {
	return s.by(&s.teams[i], &s.teams[j])
}
