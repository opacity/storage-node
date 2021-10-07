package models

import (
	"errors"

	"github.com/opacity/storage-node/utils"
)

func GetPlanInfoByID(planID uint) (planInfo utils.PlanInfo, err error) {
	err = DB.Where("id = ?", planID).Find(&planInfo).Error
	return
}

func CheckPlanInfoIsFree(planInfo utils.PlanInfo) bool {
	return planInfo.Cost == 0 && planInfo.CostInUSD == 0
}

func GetAllPlans() ([]utils.PlanInfo, error) {
	pi := []utils.PlanInfo{}
	piResults := DB.Find(&pi)

	if piResults.RowsAffected == 0 {
		return pi, errors.New("no plans added")
	}

	return pi, nil
}

func DeletePlanByID(planInfoId uint) error {
	return DB.Where("id = ?", planInfoId).Delete(utils.PlanInfo{}).Error
}
