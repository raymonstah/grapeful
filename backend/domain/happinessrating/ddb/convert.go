package ddb

import "github.com/raymonstah/grapeful/domain/happinessrating"


// Convert from the DAO event to the aggregate
func Convert(event HappinessRatingEvent) happinessrating.HappinessRating {
	return happinessrating.HappinessRating{
		UserID: event.UserID,
		Time:   event.Time,
		Score:  event.Score,
		Notes:  event.Notes,
	}
}
