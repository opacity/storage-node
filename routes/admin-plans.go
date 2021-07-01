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
	defer c.Request.Body.Close()

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
	defer c.Request.Body.Close()

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
	defer c.Request.Body.Close()

	err := c.Request.ParseForm()
	if err != nil {
		return BadRequestResponse(c, errors.New("something went wrong"))
	}

	planInfo := utils.PlanInfo{}
	storageInGBInit, err := strconv.Atoi(c.Request.PostForm["storageInGBInit"][0])
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	err = models.DB.Where("storage_in_gb = ?", storageInGBInit).First(&planInfo).Error
	if err != nil {
		return NotFoundResponse(c, errors.New("no plan found"))
	}

	cost, err := strconv.ParseFloat(c.Request.PostForm["cost"][0], 64)
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	costInUSD, err := strconv.ParseFloat(c.Request.PostForm["costInUSD"][0], 64)
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	storageInGB, err := strconv.Atoi(c.Request.PostForm["storageInGB"][0])
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	maxFolders, err := strconv.Atoi(c.Request.PostForm["maxFolders"][0])
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	maxMetadataSizeInMB, err := strconv.ParseInt(c.Request.PostForm["maxMetadataSizeInMB"][0], 10, 64)
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	planInfo.Name = c.Request.PostForm["name"][0]
	planInfo.Cost = cost
	planInfo.CostInUSD = costInUSD
	planInfo.StorageInGB = storageInGB
	planInfo.MaxFolders = maxFolders
	planInfo.MaxMetadataSizeInMB = maxMetadataSizeInMB

	if err := models.DB.Save(&planInfo).Error; err == nil {
		if plan, ok := utils.Env.Plans[storageInGBInit]; ok {
			if err != nil {
				return BadRequestResponse(c, err)
			}
			utils.Env.Plans[storageInGBInit] = planInfo

			return OkResponse(c, StatusRes{Status: "plan " + plan.Name + " was changed"})
		}
	}

	return BadRequestResponse(c, err)
}
