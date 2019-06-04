package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tryanzu/core/board/flags"
	"github.com/tryanzu/core/core/events"
	"github.com/tryanzu/core/core/user"
	"github.com/tryanzu/core/deps"
	"gopkg.in/mgo.v2/bson"
)

type upsertFlagForm struct {
	RelatedTo string        `json:"related_to" binding:"required,eq=post|eq=comment"`
	RelatedID bson.ObjectId `json:"related_id" binding:"required"`
	Category  string        `json:"category" binding:"required"`
	Content   string        `json:"content" binding:"max=255"`
}

// NewFlag godoc
// @Summary Flag an entity (post/comment)
// @Accept json
// @Produce json
// @Success 200 {object} flags.Flag
// @Header 200 {string} Authorization "Bearer $token"
// @Failure 400 {object} controller.HTTPErr
// @Failure 404 {object} controller.HTTPErr
// @Failure 500 {object} controller.HTTPErr
// @Router /flags [post]
func NewFlag(c *gin.Context) {
	var form upsertFlagForm
	if err := c.BindJSON(&form); err != nil {
		jsonBindErr(c, http.StatusBadRequest, "Invalid flag request, check parameters", err)
		return
	}

	category, err := flags.CastCategory(form.Category)
	if err != nil {
		jsonErr(c, http.StatusBadRequest, "Invalid flag category")
		return
	}

	usr := c.MustGet("user").(user.User)
	if count := flags.TodaysCountByUser(deps.Container, usr.Id); count > 10 {
		jsonErr(c, http.StatusPreconditionFailed, "Can't flag anymore for today")
		return
	}

	flag, err := flags.UpsertFlag(deps.Container, flags.Flag{
		UserID:    usr.Id,
		RelatedID: form.RelatedID,
		RelatedTo: form.RelatedTo,
		Content:   form.Content,
		Category:  category,
	})
	if err != nil {
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	events.In <- events.NewFlag(flag.ID)
	c.JSON(200, flag)
}

// Flag status request.
func Flag(c *gin.Context) {
	var (
		id      bson.ObjectId
		related = c.Params.ByName("related")
	)
	// ID validation.
	if id = bson.ObjectIdHex(c.Params.ByName("id")); !id.Valid() {
		jsonErr(c, http.StatusBadRequest, "malformed request, invalid id")
		return
	}
	usr := c.MustGet("user").(user.User)
	f, err := flags.FindOne(deps.Container, related, id, usr.Id)
	if err != nil {
		jsonErr(c, http.StatusNotFound, "flag not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"flag": f})
}
