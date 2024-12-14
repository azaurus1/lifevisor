package data

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Models struct {
	Bucket Bucket
	Event  Event
}

type Bucket struct {
	Key      int
	ID       string
	Created  time.Time
	Name     string
	Type     string
	Client   string
	Hostname string
}

type Event struct {
	ID        int
	BucketID  int
	Timestamp time.Time
	Duration  float64
	DataStr   string
}

func New(conn *pgxpool.Pool) *Models {
	repo = NewPostgresRepository(conn)
	return &Models{}
}
