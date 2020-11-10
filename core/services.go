package core

import (
	"groups/core/model"
	"log"
)

func (app *Application) applyDataProtection(current *model.User, group model.Group) map[string]interface{} {
	//1 apply data protection for "anonymous"
	if current == nil {
		return app.protectDataForAnonymous(group)
	}

	//2 apply data protection for "group admin"
	if group.IsGroupAdmin(current.ID) {
		return app.protectDataForAdmin(group)
	}

	//3 apply data protection for "group member"
	if group.IsGroupMember(current.ID) {
		//TODO
		log.Printf("%s - member", group.Title)
	}

	//4 apply data protection for "group pending"
	if group.IsGroupPending(current.ID) {
		//TODO
		log.Printf("%s - pending", group.Title)
	}

	//5 apply data protection for "NOT member"
	//TODO
	return nil
}

func (app *Application) protectDataForAnonymous(group model.Group) map[string]interface{} {
	switch group.Privacy {
	case "public":
		item := make(map[string]interface{})

		item["id"] = group.ID
		item["category"] = group.Category
		item["title"] = group.Title
		item["privacy"] = group.Privacy
		item["description"] = group.Description
		item["image_url"] = group.ImageURL
		item["web_url"] = group.WebURL
		item["members_count"] = group.MembersCount
		item["tags"] = group.Tags
		item["membership_questions"] = group.MembershipQuestions

		//members
		membersCount := len(group.Members)
		var membersItems []map[string]interface{}
		if membersCount > 0 {
			for _, current := range group.Members {
				if current.Status == "admin" || current.Status == "member" {
					mItem := make(map[string]interface{})
					mItem["id"] = current.ID
					mItem["name"] = current.Name
					mItem["email"] = current.Email
					mItem["photo_url"] = current.PhotoURL
					mItem["status"] = current.Status
					membersItems = append(membersItems, mItem)
				}
			}
		}
		item["members"] = membersItems

		item["date_created"] = group.DateCreated
		item["date_updated"] = group.DateUpdated

		//TODO add events and posts when they appear
		return item
	case "private":
		//we must protect events, posts and members(only admins are visible)
		item := make(map[string]interface{})

		item["id"] = group.ID
		item["category"] = group.Category
		item["title"] = group.Title
		item["privacy"] = group.Privacy
		item["description"] = group.Description
		item["image_url"] = group.ImageURL
		item["web_url"] = group.WebURL
		item["members_count"] = group.MembersCount
		item["tags"] = group.Tags
		item["membership_questions"] = group.MembershipQuestions

		//members
		membersCount := len(group.Members)
		var membersItems []map[string]interface{}
		if membersCount > 0 {
			for _, current := range group.Members {
				if current.Status == "admin" {
					mItem := make(map[string]interface{})
					mItem["id"] = current.ID
					mItem["name"] = current.Name
					mItem["email"] = current.Email
					mItem["photo_url"] = current.PhotoURL
					mItem["status"] = current.Status
					membersItems = append(membersItems, mItem)
				}
			}
		}
		item["members"] = membersItems

		item["date_created"] = group.DateCreated
		item["date_updated"] = group.DateUpdated

		return item
	}
	return nil
}

func (app *Application) protectDataForAdmin(group model.Group) map[string]interface{} {
	item := make(map[string]interface{})

	item["id"] = group.ID
	item["category"] = group.Category
	item["title"] = group.Title
	item["privacy"] = group.Privacy
	item["description"] = group.Description
	item["image_url"] = group.ImageURL
	item["web_url"] = group.WebURL
	item["members_count"] = group.MembersCount
	item["tags"] = group.Tags
	item["membership_questions"] = group.MembershipQuestions

	//members
	membersCount := len(group.Members)
	var membersItems []map[string]interface{}
	if membersCount > 0 {
		for _, current := range group.Members {
			mItem := make(map[string]interface{})
			mItem["id"] = current.ID
			mItem["name"] = current.Name
			mItem["email"] = current.Email
			mItem["photo_url"] = current.PhotoURL
			mItem["status"] = current.Status

			//member answers
			answersCount := len(current.MemberAnswers)
			var answersItems []map[string]interface{}
			if answersCount > 0 {
				for _, cAnswer := range current.MemberAnswers {
					aItem := make(map[string]interface{})
					aItem["question"] = cAnswer.Question
					aItem["answer"] = cAnswer.Answer
					answersItems = append(answersItems, aItem)
				}
			}
			mItem["member_answers"] = answersItems

			mItem["date_created"] = current.DateCreated
			mItem["date_updated"] = current.DateUpdated
			membersItems = append(membersItems, mItem)
		}
	}
	item["members"] = membersItems

	item["date_created"] = group.DateCreated
	item["date_updated"] = group.DateUpdated

	//TODO add events and posts when they appear
	return item
}

func (app *Application) getVersion() string {
	return app.version
}

func (app *Application) getGroupCategories() ([]string, error) {
	groupCategories, err := app.storage.ReadAllGroupCategories()
	if err != nil {
		return nil, err
	}
	return groupCategories, nil
}

func (app *Application) createGroup(current model.User, title string, description *string, category string, tags []string, privacy string,
	creatorName string, creatorEmail string, creatorPhotoURL string) (*string, error) {
	insertedID, err := app.storage.CreateGroup(title, description, category, tags, privacy,
		current.ID, creatorName, creatorEmail, creatorPhotoURL)
	if err != nil {
		return nil, err
	}
	return insertedID, nil
}

func (app *Application) getGroups(category *string) ([]map[string]interface{}, error) {
	// find the groups objects
	groups, err := app.storage.FindGroups(category)
	if err != nil {
		return nil, err
	}

	//apply data protection
	groupsList := make([]map[string]interface{}, len(groups))
	for i, item := range groups {
		groupsList[i] = app.applyDataProtection(nil, item)
	}

	return groupsList, nil
}

func (app *Application) getUserGroups(current *model.User) ([]map[string]interface{}, error) {
	// find the user groups
	groups, err := app.storage.FindUserGroups(current.ID)
	if err != nil {
		return nil, err
	}

	//apply data protection
	groupsList := make([]map[string]interface{}, len(groups))
	for i, item := range groups {
		groupsList[i] = app.applyDataProtection(current, item)
	}

	return groupsList, nil
}
