package model

//GroupsConfig represents the Groups configurations entity
type GroupsConfig struct {
	Name         string `json:"name" bson:"name"`
	UpdatePeriod int    `json:"update_period" bson:"update_period"`
}
