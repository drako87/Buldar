package gaming

import (
	"github.com/fernandez14/spartangeek-blacker/modules/user"
	"gopkg.in/mgo.v2/bson"
	"time"
	"sort"
)

func (self *Module) GetRankingBy(sort string) []RankingModel {
	
	var ranking RankingModel
	var rankings []RankingModel
	var users []RankingUserModel
	var users_id []bson.ObjectId

	database := self.Mongo.Database

	// Get the rankings with the sort parameter
	iter := database.C("stats").Find(nil).Sort("-position." + sort).Limit(50).Iter()

	for iter.Next(&ranking) {

		rankings = append(rankings, ranking)
		users_id = append(users_id, ranking.UserId)
	}

	err := database.C("users").Find(bson.M{"_id": bson.M{"$in": users_id}}).Select(bson.M{"_id": 1, "username": 1, "image": 1}).All(&users)

	if err != nil {
		panic(err)
	}

	for id, rank := range rankings {

		for _, user := range users {

			if user.Id == rank.UserId {

				rankings[id].User = user

				break
			}
		}
	}

	return rankings
}

func (self *Module) ResetGeneralRanking() {
	
	var usr user.User
	var rankings []RankingModel

	// Recover from any panic even inside this goroutine
	defer self.Errors.Recover()

	database      := self.Mongo.Database
	current_batch := time.Now()

	// Get the last batch if any
	var last_batch RankingModel
	var last_batch_available bool = false

	err := database.C("stats").Find(nil).Sort("-created_at").One(&last_batch)

	if err == nil {

		last_batch_available = true
	} 

	iter := database.C("users").Find(nil).Iter()

	for iter.Next(&usr) {

		before := RankingPositionModel{
			Wealth: 0,
			Badges: 0,
			Swords: 0,
		}

		if last_batch_available {

			var before_this RankingModel 

			err := database.C("stats").Find(bson.M{"user_id": usr.Id}).Sort("-created_at").Limit(1).One(&before_this)

			if err == nil {

				before = before_this.Before
			}
		}

		rankings = append(rankings, RankingModel{
			UserId: usr.Id,
			Badges: len(usr.Gaming.Badges),
			Swords: usr.Gaming.Swords,
			Coins:  usr.Gaming.Coins,
			Position: RankingPositionModel{
				Wealth: 0,
				Badges: 0,
				Swords: 0,
			},
			Before: before,
			Created: current_batch,
		})
	}

	sort.Sort(RankBySwords(rankings))

	for i, _ := range rankings {

		rankings[i].Position.Swords = i+1
	}

	sort.Sort(RankByCoins(rankings))

	for i, _ := range rankings {

		rankings[i].Position.Wealth = i+1
	}

	sort.Sort(RankByBadges(rankings))

	for i, _ := range rankings {

		rankings[i].Position.Badges = i+1

		err := database.C("stats").Insert(rankings[i])

		if err != nil {
			panic(err)
		}
	}
}