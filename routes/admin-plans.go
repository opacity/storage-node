package routes

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

var ErrNoPlanFound = errors.New("no plan found")

func AdminPlansGetAllHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminPlanGetAll)
}

func AdminPlansGetHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminPlanGet)
}

func AdminPlansRemoveConfirmHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminPlanRemoveConfirm)
}

func AdminPlansRemoveHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminPlanRemove)
}

func AdminPlansChangeHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminPlanChange)
}

func AdminPlansAddHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminPlanAdd)
}

func adminPlanGetAll(c *gin.Context) error {
	return adminPlanGetRenderAll(c, "")
}

func adminPlanGet(c *gin.Context) error {
	planParam, err := strconv.ParseUint(c.Param("plan"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, errors.New("something went wrong"))
	}

	plan, err := models.GetPlanInfoByID(uint(planParam))
	if err != nil {
		return NotFoundResponse(c, ErrNoPlanFound)
	}

	c.HTML(http.StatusOK, "plan-change.tmpl", gin.H{
		"title":            plan.Name,
		"plan":             plan,
		"fileStorageTypes": GetFileStorageTypesMap(),
	})
	return nil
}

func adminPlanRemoveConfirm(c *gin.Context) error {
	defer c.Request.Body.Close()

	planParam, err := strconv.ParseUint(c.Param("plan"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, errors.New("something went wrong"))
	}

	plan, err := models.GetPlanInfoByID(uint(planParam))
	if err != nil {
		return NotFoundResponse(c, ErrNoPlanFound)
	}
	c.HTML(http.StatusOK, "plan-confirm-remove.tmpl", gin.H{
		"title": plan.Name,
		"plan":  plan,
	})
	return nil
}

func adminPlanRemove(c *gin.Context) error {
	defer c.Request.Body.Close()

	planParam, err := strconv.ParseUint(c.Param("plan"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, errors.New("something went wrong"))
	}

	err = models.DeletePlanByID(uint(planParam))
	if err != nil {
		return InternalErrorResponse(c, ErrNoPlanFound)
	}

	return adminPlanGetRenderAll(c, fmt.Sprintf("Plan %d was removed", planParam))
}

func adminPlanChange(c *gin.Context) error {
	defer c.Request.Body.Close()

	err := c.Request.ParseForm()
	if err != nil {
		return BadRequestResponse(c, errors.New("something went wrong"))
	}

	planID, err := strconv.Atoi(c.Request.PostForm["ID"][0])
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	planInfo, err := models.GetPlanInfoByID(uint(planID))
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
	fileStorageType, err := strconv.ParseInt(c.Request.PostForm["fileStorageType"][0], 10, 64)
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	planInfo.Name = c.Request.PostForm["name"][0]
	planInfo.Cost = cost
	planInfo.CostInUSD = costInUSD
	planInfo.StorageInGB = storageInGB
	planInfo.MaxFolders = maxFolders
	planInfo.MaxMetadataSizeInMB = maxMetadataSizeInMB
	planInfo.MonthsInSubscription = 12
	planInfo.FileStorageType = utils.FileStorageType(fileStorageType)

	err = models.DB.Save(&planInfo).Error
	if err != nil {
		return BadRequestResponse(c, err)
	}

	return adminPlanGetRenderAll(c, fmt.Sprintf("Plan %d was updated", planInfo.ID))
}

func adminPlanAdd(c *gin.Context) error {
	defer c.Request.Body.Close()

	err := c.Request.ParseForm()
	if err != nil {
		return BadRequestResponse(c, errors.New("something went wrong"))
	}

	planInfo := utils.PlanInfo{}

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
	fileStorageType, err := strconv.ParseInt(c.Request.PostForm["fileStorageType"][0], 10, 64)
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	planInfo.Name = c.Request.PostForm["name"][0]
	planInfo.Cost = cost
	planInfo.CostInUSD = costInUSD
	planInfo.StorageInGB = storageInGB
	planInfo.MaxFolders = maxFolders
	planInfo.MaxMetadataSizeInMB = maxMetadataSizeInMB
	planInfo.MonthsInSubscription = 12
	planInfo.FileStorageType = utils.FileStorageType(fileStorageType)

	err = models.DB.Save(&planInfo).Error
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	return adminPlanGetRenderAll(c, fmt.Sprintf("Plan %s (%d) was added", planInfo.Name, planInfo.ID))
}

func adminPlanGetRenderAll(c *gin.Context, notificationMessage string) error {
	plans, err := models.GetAllPlans()
	if err != nil {
		return NotFoundResponse(c, err)
	}

	c.HTML(http.StatusOK, "plans-list.tmpl", gin.H{
		"title":                   "Change plans",
		"plans":                   plans,
		"FileStorageTypeToString": FileStorageTypeToString,
		"notificationMessage":     notificationMessage,
	})

	return nil
}

func GetFileStorageTypesMap() map[string]utils.FileStorageType {
	fileTypes := map[string]utils.FileStorageType{
		"S3":     utils.S3,
		"Sia":    utils.Sia,
		"Skynet": utils.Skynet,
	}

	return fileTypes
}

func FileStorageTypeToString(fst utils.FileStorageType) string {
	switch fst {
	case utils.S3:
		return "S3"
	case utils.Sia:
		return "Sia"
	case utils.Skynet:
		return "Skynet"
	case utils.Galaxy:
		return "Galaxy"
	default:
		return "unkown"
	}
}
