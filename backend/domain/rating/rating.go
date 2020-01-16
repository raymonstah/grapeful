package rating

// HappinessRating is a representation of how someone would rate their happiness
type HappinessRating struct {
	UserID string `validate:"required"`
	When   string `validate:"required"`
	Score  int    `validate:"min=1,max=10"`
	Notes  string
}

// RateInput is the input used to create a new rating
type RateInput struct {
	UserID string `validate:"required"`
	Score  int    `validate:"min=1,max=9"`
	Notes  string
}

// Commands allows one to rate their happiness
type Commands interface {
	Rate(rating RateInput) error
}

// GetRatingsInput is the input used to find all of a given user's ratings
type GetRatingsInput struct {
	UserID string `validate:"required"`
}

// Queries are the possible queries one can perform on ratings
type Queries interface {
	GetRatings(input GetRatingsInput) ([]HappinessRating, error)
}