package model

import "time"

//Event represents event entity
type Event struct {
	ClientID      string     `json:"client_id" bson:"client_id"`
	EventID       string     `json:"event_id" bson:"event_id"`
	GroupID       string     `json:"group_id" bson:"group_id"`
	DateCreated   time.Time  `json:"date_created" bson:"date_created"`
	Creator       Creator    `json:"creator" bson:"creator"`
	ToMembersList []ToMember `json:"to_members" bson:"to_members"` // nil or empty means everyone; non-empty means visible to those user ids and admins
} // @name Event
