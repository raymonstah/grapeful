package ddb

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	"github.com/raymonstah/grapeful/domain/happinessrating"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/assert"
	"gopkg.in/go-playground/validator.v9"
	"math/rand"
	"testing"
	"time"
)

const (
	maxScore = 10
)

func withDAO(t *testing.T, do func(ctx context.Context, dao *HappinessRatingEventDAO)) {
	s := session.Must(session.NewSession(aws.NewConfig().
		WithRegion("us-west-2").
		WithEndpoint("http://localhost:8000")))
	db := dynamo.New(s)
	tableName := fmt.Sprintf("happiness-events-%v", time.Now().Unix())
	ctx := context.Background()
	err := db.CreateTable(tableName, HappinessRatingEvent{}).RunWithContext(ctx)
	assert.Nil(t, err)
	table := db.Table(tableName)
	defer table.DeleteTable().RunWithContext(ctx)
	dao := New(tableName, db)
	do(ctx, dao)

}

func TestHappinessRatingEventDAO_Rate(t *testing.T) {
	withDAO(t, func(ctx context.Context, dao *HappinessRatingEventDAO) {
		err := dao.Rate(ctx, happinessrating.RateInput{
			UserID: ksuid.New().String(),
			Score:  10,
		})
		assert.Nil(t, err)
	})
}

func TestHappinessRatingEventDAO_Rate_BadInput(t *testing.T) {
	withDAO(t, func(ctx context.Context, dao *HappinessRatingEventDAO) {
		err := dao.Rate(ctx, happinessrating.RateInput{
			UserID: ksuid.New().String(),
			// score is missing
		})
		assert.NotNil(t, err)
		var validationErr validator.ValidationErrors
		isValidationError := errors.As(err, &validationErr)
		assert.True(t, isValidationError)
	})
}

func TestHappinessRatingEventDAO_GetRatings(t *testing.T) {
	const (
		userID = "123456"
		n      = 100
	)

	withDAO(t, func(ctx context.Context, dao *HappinessRatingEventDAO) {

		for i := 0; i < n; i++ {
			err := dao.Rate(ctx, happinessrating.RateInput{
				UserID: userID,
				Score:  getRandomScore(),
				Notes:  "",
			})
			assert.Nil(t, err)
		}

		ratings, err := dao.GetRatings(ctx, happinessrating.GetRatingsInput{UserID: userID})
		assert.Nil(t, err)
		assert.Len(t, ratings, n)

		total := 0
		for _, rating := range ratings {
			total += rating.Score
		}
		avg := total / len(ratings)
		assert.Equal(t, 5, avg, "using random scores should yield an average of 5")
	})
}

func TestHappinessRatingEventDAO_GetRatings_BadInput(t *testing.T) {

	withDAO(t, func(ctx context.Context, dao *HappinessRatingEventDAO) {

		_, err := dao.GetRatings(ctx, happinessrating.GetRatingsInput{})
		assert.NotNil(t, err)

	})
}


func TestHappinessRatingEventDAO_GetRatings_NoResults(t *testing.T) {

	withDAO(t, func(ctx context.Context, dao *HappinessRatingEventDAO) {

		ratings, err := dao.GetRatings(ctx, happinessrating.GetRatingsInput{UserID: ksuid.New().String()})
		assert.Nil(t, err)
		assert.Empty(t, ratings)
	})
}

func getRandomScore() int {
	return rand.Intn(maxScore) + 1 // because offset
}
