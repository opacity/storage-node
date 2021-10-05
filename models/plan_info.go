package models

import "github.com/opacity/storage-node/utils"

func GetPlanInfoByID(planID uint) (planInfo utils.PlanInfo, err error) {
	err = DB.Where("id = ?", planID).Find(&planInfo).Error
	return
}

func CheckPlanInfoIsFree(planInfo utils.PlanInfo) bool {
	return planInfo.Cost == 0 && planInfo.CostInUSD == 0
}
