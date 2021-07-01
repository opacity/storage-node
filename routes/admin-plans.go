package routes

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

/*
@TODO: add docs
*/
func AdminPlansEditHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminPlansEdit)
}

/*
@TODO: add docs
*/
func AdminPlansGetHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminPlanGet)
}

/*
@TODO: add docs
*/
func AdminPlansRemoveConfirmHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminPlanRemoveConfirm)
}

/*
@TODO: add docs
*/
func AdminPlansRemoveHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminPlanRemove)
}

/*
@TODO: add docs
*/
func AdminPlansChangeHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminPlanChange)
}

func adminPlansEdit(c *gin.Context) error {
	defer c.Request.Body.Close()

	return OkResponse(c, StatusRes{Status: "plan was updated"})
}

func adminPlanGet(c *gin.Context) error {
	planParam, err := strconv.Atoi(c.Param("plan"))
	if err != nil {
		return InternalErrorResponse(c, errors.New("something went wrong"))
	}
	if plan, ok := utils.Env.Plans[planParam]; ok {
		c.HTML(http.StatusOK, "plan.tmpl", gin.H{
			"title": plan.Name + " " + strconv.Itoa(plan.StorageInGB),
			"plan":  plan,
		})
		return nil
	}

	return NotFoundResponse(c, errors.New("no plan found"))
}

func adminPlanRemoveConfirm(c *gin.Context) error {
	planParam, err := strconv.Atoi(c.Param("plan"))
	if err != nil {
		return InternalErrorResponse(c, errors.New("something went wrong"))
	}
	if plan, ok := utils.Env.Plans[planParam]; ok {
		c.HTML(http.StatusOK, "plan-confirm-remove.tmpl", gin.H{
			"title": plan.Name + " " + strconv.Itoa(plan.StorageInGB),
			"plan":  plan,
		})
		return nil
	}

	return NotFoundResponse(c, errors.New("no plan found"))
}

func adminPlanRemove(c *gin.Context) error {
	planParam, err := strconv.Atoi(c.Param("plan"))
	if err != nil {
		return InternalErrorResponse(c, errors.New("something went wrong"))
	}
	if plan, ok := utils.Env.Plans[planParam]; ok {
		err := models.DB.Where("storage_in_gb = ?", plan.StorageInGB).Delete(utils.PlanInfo{}).Error
		if err != nil {
			return InternalErrorResponse(c, err)
		}

		delete(utils.Env.Plans, planParam)

		return OkResponse(c, StatusRes{Status: "plan " + plan.Name + " was removed"})
	}

	return NotFoundResponse(c, errors.New("no plan found"))
}

func adminPlanChange(c *gin.Context) error {
	planParam, err := strconv.Atoi(c.Param("plan"))
	if err != nil {
		return InternalErrorResponse(c, errors.New("something went wrong"))
	}
	if plan, ok := utils.Env.Plans[planParam]; ok {
		// get values from form
		return OkResponse(c, StatusRes{Status: "plan " + plan.Name + " was changed"})
	}

	return NotFoundResponse(c, errors.New("no plan found"))
}
