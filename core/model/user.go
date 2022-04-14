package model

import "time"

//User represents user entity
type User struct {
	ID            string     `json:"id" bson:"_id"`
	IsAnonymous   bool       `json:"is_anonymous" bson:"is_anonymous"`
	IsCoreUser    bool       `json:"is_core_user" bson:"is_core_user"`
	ExternalID    string     `json:"external_id" bson:"external_id"`
	Email         string     `json:"email" bson:"email"`
	Name          string     `json:"name" bson:"name"`
	DateCreated   time.Time  `json:"date_created" bson:"date_created"`
	DateUpdated   *time.Time `json:"date_updated" bson:"date_updated"`
	ClientID      string     `bson:"client_id"`
	OriginalToken string
} // @name User

// CoreAccount wraps the account structure from the Core BB
// @name CoreAccount
type CoreAccount struct {
	AuthTypes []struct {
		Active     bool   `json:"active"`
		Code       string `json:"code"`
		ID         string `json:"id"`
		Identifier string `json:"identifier"`
		Params     struct {
			User struct {
				Email          string        `json:"email"`
				FirstName      string        `json:"first_name"`
				Groups         []interface{} `json:"groups"`
				Identifier     string        `json:"identifier"`
				LastName       string        `json:"last_name"`
				MiddleName     string        `json:"middle_name"`
				Roles          []string      `json:"roles"`
				SystemSpecific struct {
					PreferredUsername string `json:"preferred_username"`
				} `json:"system_specific"`
			} `json:"user"`
		} `json:"params"`
	} `json:"auth_types"`
	Groups      []interface{} `json:"groups"`
	ID          string        `json:"id"`
	Permissions []interface{} `json:"permissions"`
	Preferences struct {
		Favorites interface{} `json:"favorites"`
		Interests struct {
		} `json:"interests"`
		PrivacyLevel int      `json:"privacy_level"`
		Roles        []string `json:"roles"`
		Settings     struct {
		} `json:"settings"`
		Tags struct {
		} `json:"tags"`
		Voter struct {
			RegisteredVoter bool        `json:"registered_voter"`
			VotePlace       string      `json:"vote_place"`
			Voted           bool        `json:"voted"`
			VoterByMail     interface{} `json:"voter_by_mail"`
		} `json:"voter"`
	} `json:"preferences"`
	Profile struct {
		Address   string `json:"address"`
		BirthYear int    `json:"birth_year"`
		Country   string `json:"country"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		ID        string `json:"id"`
		LastName  string `json:"last_name"`
		Phone     string `json:"phone"`
		PhotoURL  string `json:"photo_url"`
		State     string `json:"state"`
		ZipCode   string `json:"zip_code"`
	} `json:"profile"`
	Roles []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"roles"`
}
