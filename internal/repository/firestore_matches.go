package repository

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/guilfoos/crew-predictions/internal/models"
)

type FirestoreMatchStore struct {
	client *firestore.Client
}

func NewFirestoreMatchStore(projectID string) (*FirestoreMatchStore, error) {
	client, err := firestore.NewClient(context.Background(), projectID)
	if err != nil {
		return nil, err
	}
	return &FirestoreMatchStore{client: client}, nil
}

func (s *FirestoreMatchStore) SaveAll(matches []models.Match) error {
	ctx := context.Background()
	batch := s.client.BulkWriter(ctx)
	for _, m := range matches {
		ref := s.client.Collection("matches").Doc(m.ID)
		events := make([]map[string]any, len(m.Events))
		for i, e := range m.Events {
			events[i] = map[string]any{
				"clock":   e.Clock,
				"typeID":  e.TypeID,
				"team":    e.Team,
				"players": e.Players,
			}
		}
		_, _ = batch.Set(ref, map[string]any{
			"homeTeam":       m.HomeTeam,
			"awayTeam":       m.AwayTeam,
			"kickoff":        m.Kickoff,
			"status":         m.Status,
			"homeScore":      m.HomeScore,
			"awayScore":      m.AwayScore,
			"state":          m.State,
			"displayClock":   m.DisplayClock,
			"venue":          m.Venue,
			"homeRecord":     m.HomeRecord,
			"awayRecord":     m.AwayRecord,
			"homeForm":       m.HomeForm,
			"awayForm":       m.AwayForm,
			"homeLogo":       m.HomeLogo,
			"awayLogo":       m.AwayLogo,
			"attendance":     m.Attendance,
			"referee":        m.Referee,
			"events":         events,
			"lastPollAt":     m.LastPollAt,
			"chainSeededFor": m.ChainSeededFor,
			"abandonedAt":    m.AbandonedAt,
		})
	}
	batch.Flush()
	return nil
}

func (s *FirestoreMatchStore) GetAll() ([]models.Match, error) {
	ctx := context.Background()
	snaps, err := s.client.Collection("matches").Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	out := make([]models.Match, 0, len(snaps))
	for _, snap := range snaps {
		m, err := toMatch(snap)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *FirestoreMatchStore) Reset() {}

func toMatch(snap *firestore.DocumentSnapshot) (models.Match, error) {
	var doc struct {
		HomeTeam       string    `firestore:"homeTeam"`
		AwayTeam       string    `firestore:"awayTeam"`
		Kickoff        time.Time `firestore:"kickoff"`
		Status         string    `firestore:"status"`
		HomeScore      string    `firestore:"homeScore"`
		AwayScore      string    `firestore:"awayScore"`
		State          string    `firestore:"state"`
		DisplayClock   string    `firestore:"displayClock"`
		Venue          string    `firestore:"venue"`
		HomeRecord     string    `firestore:"homeRecord"`
		AwayRecord     string    `firestore:"awayRecord"`
		HomeForm       string    `firestore:"homeForm"`
		AwayForm       string    `firestore:"awayForm"`
		HomeLogo       string    `firestore:"homeLogo"`
		AwayLogo       string    `firestore:"awayLogo"`
		Attendance     int64     `firestore:"attendance"`
		Referee        string    `firestore:"referee"`
		LastPollAt     time.Time `firestore:"lastPollAt"`
		ChainSeededFor time.Time `firestore:"chainSeededFor"`
		AbandonedAt    time.Time `firestore:"abandonedAt"`
		Events         []struct {
			Clock   string   `firestore:"clock"`
			TypeID  string   `firestore:"typeID"`
			Team    string   `firestore:"team"`
			Players []string `firestore:"players"`
		} `firestore:"events"`
	}
	if err := snap.DataTo(&doc); err != nil {
		return models.Match{}, err
	}
	events := make([]models.MatchEvent, len(doc.Events))
	for i, e := range doc.Events {
		events[i] = models.MatchEvent{
			Clock:   e.Clock,
			TypeID:  e.TypeID,
			Team:    e.Team,
			Players: e.Players,
		}
	}
	return models.Match{
		ID:             snap.Ref.ID,
		HomeTeam:       doc.HomeTeam,
		AwayTeam:       doc.AwayTeam,
		Kickoff:        doc.Kickoff,
		Status:         doc.Status,
		HomeScore:      doc.HomeScore,
		AwayScore:      doc.AwayScore,
		State:          doc.State,
		DisplayClock:   doc.DisplayClock,
		Venue:          doc.Venue,
		HomeRecord:     doc.HomeRecord,
		AwayRecord:     doc.AwayRecord,
		HomeForm:       doc.HomeForm,
		AwayForm:       doc.AwayForm,
		HomeLogo:       doc.HomeLogo,
		AwayLogo:       doc.AwayLogo,
		Attendance:     int(doc.Attendance),
		Referee:        doc.Referee,
		LastPollAt:     doc.LastPollAt,
		ChainSeededFor: doc.ChainSeededFor,
		AbandonedAt:    doc.AbandonedAt,
		Events:         events,
	}, nil
}
