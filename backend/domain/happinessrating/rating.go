package happinessrating

import (
	"context"
	"time"
)

// HappinessRating is a representation of how someone would rate their happiness
type HappinessRating struct {
	UserID string
	Time   time.Time
	Score  int
	Notes  string
}

// RateInput is the input used to create a new rating
type RateInput struct {
	UserID string `validate:"required"`
	Score  int    `validate:"min=1,max=10"`
	Notes  string
}

// Mutator allows one to rate their happiness
type Mutator interface {
	Rate(ctx context.Context, rating RateInput) error
}

// GetRatingsInput is the input used to find all of a given user's ratings
type GetRatingsInput struct {
	UserID string `validate:"required"`
}

// Finder are the possible queries one can perform on ratings
type Finder interface {
	GetRatings(ctx context.Context, input GetRatingsInput) ([]HappinessRating, error)
}
