package ddb

import (
	"context"
	"fmt"
	"github.com/guregu/dynamo"
	"github.com/raymonstah/grapeful/backend/domain/happinessrating"
	"gopkg.in/go-playground/validator.v9"
	"time"
)

var validate = validator.New()

// HappinessRatingEvent is the event stored in the event store
// It is considered the source of truth
type HappinessRatingEvent struct {
	UserID string    `dynamo:"user_id,hash"`
	Time   time.Time `dynamo:"time,range"`
	Score  int       `dynamo:"score"`
	Notes  string    `dynamo:"notes,omitempty"`
}

// HappinessRatingEventDAO is the access layer to the happiness events
type HappinessRatingEventDAO struct {
	table dynamo.Table
}

// New returns a new DAO to access happiness events
func New(ratingTableName string, db *dynamo.DB) *HappinessRatingEventDAO {
	return &HappinessRatingEventDAO{
		table: db.Table(ratingTableName),
	}
}

// Rate allows one to rate their happiness
func (d *HappinessRatingEventDAO) Rate(ctx context.Context, rating happinessrating.RateInput) error {
	if err := validate.Struct(rating); err != nil {
		return err
	}
	event := HappinessRatingEvent{
		UserID: rating.UserID,
		Time:   time.Now(),
		Score:  rating.Score,
		Notes:  rating.Notes,
	}
	if err := d.table.Put(event).RunWithContext(ctx); err != nil {
		return fmt.Errorf("error putting new happiness event %+v: %w", event, err)
	}

	return nil
}

// GetRatings gets all ratings by partition key
func (d *HappinessRatingEventDAO) GetRatings(ctx context.Context, input happinessrating.GetRatingsInput) (results []happinessrating.HappinessRating, err error) {
	if err := validate.Struct(input); err != nil {
		return nil, err
	}

	var ratings []HappinessRatingEvent
	if err := d.table.Get("user_id", input.UserID).AllWithContext(ctx, &ratings); err != nil {
		return nil, fmt.Errorf("error getting all ratings for user %+v: %w", input.UserID, err)
	}

	// convert results
	for _, rating := range ratings {
		results = append(results, Convert(rating))
	}

	return results, nil
}
