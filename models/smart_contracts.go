package models

/* Defines the smart contracts address and related data */
type SmartContracts struct {
	ID      uint   `gorm:"primary_key" json:"smartContractId"`
	Network string `json:"network"`
	Address string `json:"address"`
}
