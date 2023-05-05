package detector

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type jsConfig struct {
	SiteName string `json:"siteName"`
}

func createTestPlayer() model.PlayerCollection {
	var randPlayer = func(userId int64) *model.Player {
		team := model.Blu
		if userId%2 == 0 {
			team = model.Red
		}
		sid := steamid.SID64(76561197960265728 + userId)
		return &model.Player{
			SteamIdString:    sid.String(),
			Name:             golib.RandomString(40),
			CreatedOn:        time.Now(),
			UpdatedOn:        time.Now(),
			ProfileUpdatedOn: time.Now(),
			KillsOn:          rand.Intn(20),
			RageQuits:        rand.Intn(10),
			DeathsBy:         rand.Intn(20),
			Notes:            "User notes \ngo here",
			Whitelisted:      false,
			RealName:         "Real Name Goes Here",
			NamePrevious:     "",
			AccountCreatedOn: time.Time{},
			Visibility:       0,
			AvatarHash:       "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb",
			CommunityBanned:  false,
			NumberOfVACBans:  0,
			LastVACBanOn:     nil,
			NumberOfGameBans: 0,
			EconomyBan:       false,
			Team:             team,
			Connected:        float64(rand.Intn(3600)),
			UserId:           userId,
			Ping:             rand.Intn(150),
			Kills:            rand.Intn(50),
			Deaths:           rand.Intn(300),
			Matches:          []*rules.MatchResult{},
		}
	}
	var testPlayers model.PlayerCollection
	for i := int64(0); i < 24; i++ {
		p := randPlayer(i)
		switch i {
		case 1:
			p.NumberOfVACBans = 2
			last := time.Now().AddDate(-1, 0, 0)
			p.LastVACBanOn = &last
		case 4:
			p.Matches = append(p.Matches, &rules.MatchResult{
				Origin:      "Test Rules List",
				Attributes:  []string{"cheater"},
				MatcherType: "string",
			})
		case 6:
			p.Matches = append(p.Matches, &rules.MatchResult{
				Origin:      "Test Rules List",
				Attributes:  []string{"other"},
				MatcherType: "string",
			})

		case 7:
			p.Team = model.Spec
		}
		testPlayers = append(testPlayers, p)
	}
	return testPlayers
}

func getPlayers() gin.HandlerFunc {
	testPlayers := createTestPlayer()
	return func(ctx *gin.Context) {
		if _, isTest := os.LookupEnv("TEST"); isTest {
			responseOK(ctx, http.StatusOK, testPlayers)
			return
		}
		playersMu.RLock()
		defer playersMu.RUnlock()
		p := model.PlayerCollection{}
		if players != nil {
			p = players
		}
		responseOK(ctx, http.StatusOK, p)
	}
}

func postMarkPlayer() gin.HandlerFunc {
	type postOpts struct {
		SteamID steamid.SID64 `json:"steamID"`
		Attrs   []string      `json:"attrs"`
	}
	return func(ctx *gin.Context) {

	}
}

func responseErr(ctx *gin.Context, status int, data any) {
	ctx.JSON(status, data)
}

func responseOK(ctx *gin.Context, status int, data any) {
	ctx.JSON(status, data)
}

func bind(ctx *gin.Context, receiver any) bool {
	if errBind := ctx.BindJSON(&receiver); errBind != nil {
		responseErr(ctx, http.StatusBadRequest, gin.H{
			"error": "Invalid request parameters",
		})
		return false
	}
	return true
}

func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			logger.Error("Unhandled HTTP Error", zap.Error(ginErr))
		}
	}
}
