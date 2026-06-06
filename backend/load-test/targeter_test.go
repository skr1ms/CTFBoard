package load_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sync/atomic"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

type roundRobinIndex struct {
	n atomic.Uint64
}

func (r *roundRobinIndex) next(n int) int {
	if n <= 0 {
		return 0
	}

	return int(r.n.Add(1)-1) % n
}

func ChallengeListTargeter(f *TestFixture) vegeta.Targeter {
	rr := &roundRobinIndex{}

	return func(t *vegeta.Target) error {
		u := f.Users[rr.next(len(f.Users))]
		t.Method = http.MethodGet
		t.URL = f.BaseURL + "/api/v1/challenges"
		t.Header = http.Header{
			"Authorization": {u.Token},
		}

		return nil
	}
}

func ScoreboardTargeter(f *TestFixture) vegeta.Targeter {
	rr := &roundRobinIndex{}

	return func(t *vegeta.Target) error {
		u := f.Users[rr.next(len(f.Users))]
		t.Method = http.MethodGet
		t.URL = f.BaseURL + "/api/v1/scoreboard"
		t.Header = http.Header{
			"Authorization": {u.Token},
		}

		return nil
	}
}

type submitBody struct {
	Flag string `json:"flag"`
}

func SubmitWrongFlagTargeter(f *TestFixture) vegeta.Targeter {
	rr := &roundRobinIndex{}

	return func(t *vegeta.Target) error {
		u := f.Users[rr.next(len(f.Users))]
		chalIdx := rand.IntN(len(f.ChallengeIDs))
		chalID := f.ChallengeIDs[chalIdx]

		body, err := json.Marshal(submitBody{Flag: fmt.Sprintf("WRONG{flag_%d}", rand.IntN(1_000_000))})
		if err != nil {
			return err
		}

		t.Method = http.MethodPost
		t.URL = f.BaseURL + "/api/v1/challenges/" + chalID + "/submit"
		t.Header = http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {u.Token},
		}
		t.Body = body

		return nil
	}
}

func LoginTargeter(f *TestFixture, emails, passwords []string) vegeta.Targeter {
	rr := &roundRobinIndex{}

	return func(t *vegeta.Target) error {
		idx := rr.next(len(emails))

		body, err := json.Marshal(map[string]string{
			"email":    emails[idx],
			"password": passwords[idx],
		})
		if err != nil {
			return err
		}

		t.Method = http.MethodPost
		t.URL = f.BaseURL + "/api/v1/auth/login"
		t.Header = http.Header{
			"Content-Type": {"application/json"},
		}
		t.Body = body

		return nil
	}
}

func MeTargeter(f *TestFixture) vegeta.Targeter {
	rr := &roundRobinIndex{}

	return func(t *vegeta.Target) error {
		u := f.Users[rr.next(len(f.Users))]
		t.Method = http.MethodGet
		t.URL = f.BaseURL + "/api/v1/auth/me"
		t.Header = http.Header{
			"Authorization": {u.Token},
		}

		return nil
	}
}

type weightedEntry struct {
	weight int
	fn     vegeta.Targeter
}

func WeightedTargeter(entries []weightedEntry) vegeta.Targeter {
	total := 0

	for _, e := range entries {
		total += e.weight
	}

	return func(t *vegeta.Target) error {
		pick := rand.IntN(total)
		cumulative := 0

		for _, e := range entries {
			cumulative += e.weight
			if pick < cumulative {
				return e.fn(t)
			}
		}

		return entries[len(entries)-1].fn(t)
	}
}

func BruteForceTargeter(f *TestFixture, userIdx, chalIdx int) vegeta.Targeter {
	token := f.Users[userIdx%len(f.Users)].Token
	chalID := f.ChallengeIDs[chalIdx%len(f.ChallengeIDs)]

	var seq atomic.Uint64

	return func(t *vegeta.Target) error {
		n := seq.Add(1)

		body, err := json.Marshal(submitBody{Flag: fmt.Sprintf("BRUTE{attempt_%d}", n)})
		if err != nil {
			return err
		}

		t.Method = http.MethodPost
		t.URL = f.BaseURL + "/api/v1/challenges/" + chalID + "/submit"
		t.Header = http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {token},
		}
		t.Body = body

		return nil
	}
}

func MixedCTFTargeter(f *TestFixture, emails, passwords []string) vegeta.Targeter {
	return WeightedTargeter([]weightedEntry{
		{50, ChallengeListTargeter(f)},
		{20, SubmitWrongFlagTargeter(f)},
		{15, ScoreboardTargeter(f)},
		{10, LoginTargeter(f, emails, passwords)},
		{5, MeTargeter(f)},
	})
}

var registerCounter atomic.Uint64

// RegisterTargeter generates unique registration requests per call to avoid duplicate-key errors.
func RegisterTargeter(f *TestFixture) vegeta.Targeter {
	return func(t *vegeta.Target) error {
		n := registerCounter.Add(1)
		username := fmt.Sprintf("lt_reg_%d", n)
		email := fmt.Sprintf("lt_reg_%d@loadtest.local", n)
		password := "ValidPass1"

		body, err := json.Marshal(map[string]string{
			"username": username,
			"email":    email,
			"password": password,
		})
		if err != nil {
			return err
		}

		t.Method = http.MethodPost
		t.URL = f.BaseURL + "/api/v1/auth/register"
		t.Header = http.Header{
			"Content-Type": {"application/json"},
		}
		t.Body = body

		return nil
	}
}

// StatisticsTargeter round-robins over general statistics endpoints (authenticated).
func StatisticsTargeter(f *TestFixture) vegeta.Targeter {
	userRR := &roundRobinIndex{}
	epRR := &roundRobinIndex{}

	endpoints := []string{
		"/api/v1/statistics/general",
		"/api/v1/statistics/challenges",
		"/api/v1/statistics/scoreboard",
	}

	return func(t *vegeta.Target) error {
		u := f.Users[userRR.next(len(f.Users))]
		ep := endpoints[epRR.next(len(endpoints))]

		t.Method = http.MethodGet
		t.URL = f.BaseURL + ep
		t.Header = http.Header{
			"Authorization": {u.Token},
		}

		return nil
	}
}

// HintUnlockTargeter round-robins users and hint entries for the unlock endpoint.
func HintUnlockTargeter(f *TestFixture) vegeta.Targeter {
	userRR := &roundRobinIndex{}
	hintRR := &roundRobinIndex{}

	return func(t *vegeta.Target) error {
		u := f.Users[userRR.next(len(f.Users))]
		entry := f.HintEntries[hintRR.next(len(f.HintEntries))]

		t.Method = http.MethodPost
		t.URL = f.BaseURL + "/api/v1/challenges/" + entry.ChallengeID + "/hints/" + entry.HintID + "/unlock"
		t.Header = http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {u.Token},
		}
		t.Body = []byte("{}")

		return nil
	}
}

// TeamCreateTargeter generates unique team-creation requests per call.
func TeamCreateTargeter(f *TestFixture) vegeta.Targeter {
	rr := &roundRobinIndex{}

	var teamCounter atomic.Uint64

	return func(t *vegeta.Target) error {
		u := f.Users[rr.next(len(f.Users))]
		n := teamCounter.Add(1)

		body, err := json.Marshal(map[string]string{
			"name": fmt.Sprintf("lt_team_%d", n),
		})
		if err != nil {
			return err
		}

		t.Method = http.MethodPost
		t.URL = f.BaseURL + "/api/v1/teams"
		t.Header = http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {u.Token},
		}
		t.Body = body

		return nil
	}
}

// ScoreboardGraphTargeter targets the scoreboard graph endpoint.
func ScoreboardGraphTargeter(f *TestFixture) vegeta.Targeter {
	rr := &roundRobinIndex{}

	return func(t *vegeta.Target) error {
		u := f.Users[rr.next(len(f.Users))]
		t.Method = http.MethodGet
		t.URL = f.BaseURL + "/api/v1/scoreboard/graph"
		t.Header = http.Header{
			"Authorization": {u.Token},
		}

		return nil
	}
}
