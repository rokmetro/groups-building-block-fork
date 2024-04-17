// Copyright 2022 Board of Trustees of the University of Illinois.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"fmt"
	"groups/driven/rewards"
	"groups/driven/storage"
	"groups/utils"
	"time"

	"github.com/google/uuid"

	"groups/core/model"
	"groups/driven/notifications"
	"log"

	"strings"
)

const (
	defaultConfigSyncTimeout   = 60
	maxEmbeddedMemberGroupSize = 10000
	authmanUserBatchSize       = 5000
)

/*
func (app *Application) applyDataProtection(current *model.User, group model.Group) model.Group {
	//1 apply data protection for "anonymous"
	if current == nil || current.IsAnonymous {
		group.Members = []model.Member{}
	} else {
		member := group.GetMemberByUserID(current.ID)
		if member != nil && (member.IsRejected() || member.IsPendingMember()) {
			group.Members = []model.Member{}
			group.Members = append(group.Members, *member)
		}
	}
	return group
}*/

func (app *Application) getVersion() string {
	return app.version
}

func (app *Application) getGroupEntity(clientID string, id string) (*model.Group, error) {
	group, err := app.storage.FindGroup(nil, clientID, id, nil)
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (app *Application) getGroupEntityByTitle(clientID string, title string) (*model.Group, error) {
	group, err := app.storage.FindGroupByTitle(clientID, title)
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (app *Application) isGroupAdmin(clientID string, groupID string, userID string) (bool, error) {
	membership, err := app.storage.FindGroupMembership(clientID, groupID, userID)
	if err != nil {
		return false, err
	}
	if membership == nil || membership.Status != "admin" {
		return false, nil
	}

	return true, nil
}

func (app *Application) createGroup(clientID string, current *model.User, group *model.Group) (*string, *utils.GroupError) {

	var groupError *utils.GroupError
	var groupID *string
	err := app.storage.PerformTransaction(func(context storage.TransactionContext) error {
		var err error
		groupID, groupError = app.storage.CreateGroup(context, clientID, current, group, nil)
		if err != nil {
			return err
		}
		if group.ResearchGroup {
			searchParams := app.formatCoreAccountSearchParams(group.ResearchProfile)

			list := []notifications.Recipient{}
			account, err := app.corebb.GetAccounts(searchParams, &current.AppID, &current.OrgID, nil, nil)
			if err != nil {
				return nil
			}
			for _, u := range account {
				id := u.ID
				mute := false
				ne := notifications.Recipient{UserID: id, Mute: mute}
				list = append(list, ne)
			}

			app.notifications.SendNotification(list, nil, "A new research project is available", fmt.Sprintf("%s by %s", group.Title, current.Name),
				map[string]string{
					"type":        "group",
					"operation":   "research_group",
					"entity_type": "group",
					"entity_id":   group.ID,
					"entity_name": group.Title,
				},
				current.AppID,
				current.OrgID)

		}

		return nil
	})

	handleRewardsAsync := func(clientID, userID string) {
		count, grErr := app.storage.FindUserGroupsCount(clientID, current.ID)
		if grErr != nil {
			log.Printf("Error createGroup(): %s", grErr)
		} else {
			if count != nil && *count == 1 {
				app.rewards.CreateUserReward(current.ID, rewards.GroupsUserCreatedFirstGroup, "")
			}
		}
	}
	go handleRewardsAsync(clientID, current.ID)

	if groupError != nil {
		return nil, groupError
	}
	if err != nil {
		log.Printf("app.createGroup() error %s", err)
		return nil, utils.NewServerError()
	}

	return groupID, nil
}

func (app *Application) updateGroup(clientID string, current *model.User, group *model.Group) *utils.GroupError {

	err := app.storage.UpdateGroup(nil, clientID, current, group)
	if err != nil {
		return err
	}
	return nil
}

func (app *Application) updateGroupDateUpdated(clientID string, groupID string) error {
	err := app.storage.UpdateGroupDateUpdated(clientID, groupID)
	if err != nil {
		return err
	}
	return nil
}

func (app *Application) deleteGroup(clientID string, current *model.User, id string) error {
	err := app.storage.DeleteGroup(nil, clientID, id)
	if err != nil {
		return err
	}
	return nil
}

func (app *Application) getGroups(clientID string, current *model.User, filter model.GroupsFilter) ([]model.Group, error) {
	var userID *string
	if current != nil {
		userID = &current.ID
	}
	// find the groups objects
	groups, err := app.storage.FindGroups(clientID, userID, filter)
	if err != nil {
		return nil, err
	}

	return groups, nil
}

func (app *Application) getAllGroups(clientID string) ([]model.Group, error) {
	// find the groups objects
	groups, err := app.storage.FindGroups(clientID, nil, model.GroupsFilter{})
	if err != nil {
		return nil, err
	}

	return groups, nil
}

func (app *Application) getUserGroups(clientID string, current *model.User, filter model.GroupsFilter) ([]model.Group, error) {
	// find the user groups
	groups, err := app.storage.FindUserGroups(clientID, current.ID, filter)
	if err != nil {
		return nil, err
	}

	return groups, nil
}

func (app *Application) deleteUser(clientID string, current *model.User) error {
	return app.storage.DeleteUser(clientID, current.ID)
}

func (app *Application) getGroup(clientID string, current *model.User, id string) (*model.Group, error) {
	// find the group
	var userID *string
	if current != nil {
		userID = &current.ID
	}

	group, err := app.storage.FindGroup(nil, clientID, id, userID)
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (app *Application) applyMembershipApproval(clientID string, current *model.User, membershipID string, approve bool, rejectReason string) error {
	membership, err := app.storage.ApplyMembershipApproval(clientID, membershipID, approve, rejectReason)
	if err != nil {
		return fmt.Errorf("error applying membership approval: %s", err)
	}
	if err == nil && membership != nil {
		group, _ := app.storage.FindGroup(nil, clientID, membership.GroupID, nil)
		topic := "group.invitations"
		groupStr := "Group"
		if group.ResearchGroup {
			groupStr = "Research Project"
		}
		if approve {
			app.notifications.SendNotification(
				[]notifications.Recipient{
					membership.ToNotificationRecipient(membership.NotificationsPreferences.OverridePreferences &&
						(membership.NotificationsPreferences.InvitationsMuted || membership.NotificationsPreferences.AllMute)),
				},
				&topic,
				fmt.Sprintf("%s - %s", groupStr, group.Title),
				fmt.Sprintf("Your membership in '%s' %s has been approved", group.Title, strings.ToLower(groupStr)),
				map[string]string{
					"type":        "group",
					"operation":   "membership_approve",
					"entity_type": "group",
					"entity_id":   group.ID,
					"entity_name": group.Title,
				},
				current.AppID,
				current.OrgID,
			)
		} else {
			app.notifications.SendNotification(
				[]notifications.Recipient{
					membership.ToNotificationRecipient(membership.NotificationsPreferences.OverridePreferences &&
						(membership.NotificationsPreferences.InvitationsMuted || membership.NotificationsPreferences.AllMute)),
				},
				&topic,
				fmt.Sprintf("%s - %s", groupStr, group.Title),
				fmt.Sprintf("Your membership in '%s' %s has been rejected with a reason: %s", group.Title, strings.ToLower(groupStr), rejectReason),
				map[string]string{
					"type":        "group",
					"operation":   "membership_reject",
					"entity_type": "group",
					"entity_id":   group.ID,
					"entity_name": group.Title,
				},
				current.AppID,
				current.OrgID,
			)
		}

		if approve && group.CanJoinAutomatically && group.AuthmanEnabled && membership.ExternalID != "" {
			err := app.authman.AddAuthmanMemberToGroup(*group.AuthmanGroup, membership.ExternalID)
			if err != nil {
				log.Printf("err app.applyMembershipApproval() - error storing member in Authman: %s", err)
			}
		}
	} else {
		log.Printf("Unable to retrieve group by membership id: %s\n", err)
		// return err // No reason to fail if the main part succeeds
	}

	return nil
}

func (app *Application) updateMembership(clientID string, current *model.User, membershipID string, status *string, dateAttended *time.Time, notificationsPreferences *model.NotificationsPreferences) error {
	membership, _ := app.storage.FindGroupMembershipByID(clientID, membershipID)
	if membership != nil {
		if status != nil && membership.Status != *status {
			membership.Status = *status
		}
		if dateAttended != nil && membership.DateAttended == nil {
			membership.DateAttended = dateAttended
		}
		if notificationsPreferences != nil {
			membership.NotificationsPreferences = *notificationsPreferences
		}

		err := app.storage.UpdateMembership(clientID, current, membershipID, membership)
		if err != nil {
			return err
		}
	}

	return nil
}

func (app *Application) getEvents(clientID string, current *model.User, groupID string, filterByToMembers bool) ([]model.Event, error) {
	events, err := app.storage.FindEvents(clientID, current, groupID, filterByToMembers)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (app *Application) createEvent(clientID string, current *model.User, eventID string, group *model.Group, toMemberList []model.ToMember, creator *model.Creator) (*model.Event, error) {
	var skipUserID *string

	if current != nil && creator == nil {
		creator = &model.Creator{
			UserID: current.ID,
			Name:   current.Name,
			Email:  current.Email,
		}
	}
	if creator != nil {
		skipUserID = &creator.UserID
	}

	event, err := app.storage.CreateEvent(clientID, eventID, group.ID, toMemberList, creator)
	if err != nil {
		return nil, err
	}

	var userIDs []string
	var recipients []notifications.Recipient
	if len(event.ToMembersList) > 0 {
		userIDs = event.GetMembersAsUserIDs(skipUserID)
	}

	result, _ := app.storage.FindGroupMemberships(clientID, model.MembershipFilter{
		GroupIDs: []string{group.ID},
		UserIDs:  userIDs,
		Statuses: []string{"member", "admin"},
	})
	recipients = result.GetMembersAsNotificationRecipients(func(member model.GroupMembership) (bool, bool) {
		return member.IsAdminOrMember() && (skipUserID == nil || *skipUserID != member.UserID),
			member.NotificationsPreferences.OverridePreferences &&
				(member.NotificationsPreferences.EventsMuted || member.NotificationsPreferences.AllMute)
	})

	if len(recipients) > 0 {
		topic := "group.events"
		appID := app.config.AppID
		orgID := app.config.OrgID
		if current != nil {
			appID = current.AppID
			orgID = current.OrgID
		}
		groupStr := "Group"
		if group.ResearchGroup {
			groupStr = "Research Project"
		}

		go app.notifications.SendNotification(
			recipients,
			&topic,
			fmt.Sprintf("%s - %s", groupStr, group.Title),
			fmt.Sprintf("New event has been published in '%s' %s", group.Title, strings.ToLower(groupStr)),
			map[string]string{
				"type":        "group",
				"operation":   "event_created",
				"entity_type": "group",
				"entity_id":   group.ID,
				"entity_name": group.Title,
			},
			appID,
			orgID,
		)
	}

	return event, nil
}

func (app *Application) updateEvent(clientID string, _ *model.User, eventID string, groupID string, toMemberList []model.ToMember) error {
	return app.storage.UpdateEvent(clientID, eventID, groupID, toMemberList)
}

func (app *Application) deleteEvent(clientID string, _ *model.User, eventID string, groupID string) error {
	err := app.storage.DeleteEvent(clientID, eventID, groupID)
	if err != nil {
		return err
	}
	return nil
}

func (app *Application) getPosts(clientID string, current *model.User, groupID string, filterPrivatePostsValue *bool, filterByToMembers bool, postType *string, offset *int64, limit *int64, order *string) ([]*model.Post, error) {
	return app.storage.FindPosts(clientID, current, groupID, filterPrivatePostsValue, filterByToMembers, postType, offset, limit, order)
}

func (app *Application) getPost(clientID string, userID *string, groupID string, postID string, skipMembershipCheck bool, filterByToMembers bool) (*model.Post, error) {
	return app.storage.FindPost(nil, clientID, userID, groupID, postID, skipMembershipCheck, filterByToMembers)
}

func (app *Application) getUserPostCount(clientID string, userID string) (*int64, error) {
	return app.storage.GetUserPostCount(clientID, userID)
}

func (app *Application) createPost(clientID string, current *model.User, post *model.Post, group *model.Group) (*model.Post, error) {

	if group.Settings != nil && !group.Settings.PostPreferences.CanSendPostToAdmins {
		userIDs := post.GetMembersAsUserIDs(&current.ID)
		memberships, err := app.Services.FindGroupMemberships(clientID, model.MembershipFilter{
			GroupIDs: []string{post.GroupID},
			UserIDs:  userIDs,
			Statuses: []string{"admin"},
		})
		if err != nil {
			return nil, err
		}

		if len(memberships.Items) > 0 {
			var toMembers []model.ToMember
			for _, membership := range memberships.Items {
				toMembers = append(toMembers, model.ToMember{
					UserID: membership.UserID,
					Name:   membership.Name,
				})
			}
			post.ToMembersList = toMembers
		}
	}

	post, err := app.storage.CreatePost(clientID, current, post)
	if err != nil {
		return nil, err
	}

	handleRewardsAsync := func(clientID, userID string) {
		count, grErr := app.storage.GetUserPostCount(clientID, current.ID)
		if grErr != nil {
			log.Printf("Error createPost(): %s", grErr)
		} else if count != nil {
			if *count > 1 {
				app.rewards.CreateUserReward(current.ID, rewards.GroupsUserSubmittedPost, "")
			} else if *count == 1 {
				app.rewards.CreateUserReward(current.ID, rewards.GroupsUserSubmittedFirstPost, "")
			}
		}
	}
	go handleRewardsAsync(clientID, current.ID)

	handleNotification := func() {

		recipientsUserIDs, _ := app.getPostNotificationRecipientsAsUserIDs(clientID, post, &current.ID)

		result, _ := app.storage.FindGroupMemberships(clientID, model.MembershipFilter{
			GroupIDs: []string{group.ID},
			UserIDs:  recipientsUserIDs,
			Statuses: []string{"member", "admin"},
		})
		recipients := result.GetMembersAsNotificationRecipients(func(member model.GroupMembership) (bool, bool) {
			return member.IsAdminOrMember() && (current.ID != member.UserID),
				member.NotificationsPreferences.OverridePreferences &&
					(member.NotificationsPreferences.PostsMuted || member.NotificationsPreferences.AllMute)
		})

		if len(recipients) > 0 {
			groupStr := "Group"
			if group.ResearchGroup {
				groupStr = "Research Project"
			}
			title := fmt.Sprintf("%s - %s", groupStr, group.Title)
			body := fmt.Sprintf("New post has been published in '%s' %s", group.Title, strings.ToLower(groupStr))
			if post.UseAsNotification {
				title = post.Subject
				body = post.Body
			}

			topic := "group.posts"
			app.notifications.SendNotification(
				recipients,
				&topic,
				title,
				body,
				map[string]string{
					"type":         "group",
					"operation":    "post_created",
					"entity_type":  "group",
					"entity_id":    group.ID,
					"entity_name":  group.Title,
					"post_id":      post.ID,
					"post_subject": post.Subject,
					"post_body":    post.Body,
				},
				current.AppID,
				current.OrgID,
			)
		}
	}
	go handleNotification()

	return post, nil
}

func (app *Application) getPostNotificationRecipientsAsUserIDs(clientID string, post *model.Post, skipUserID *string) ([]string, error) {
	if post == nil {
		return nil, nil
	}

	if len(post.ToMembersList) > 0 {
		return post.GetMembersAsUserIDs(skipUserID), nil
	}

	var err error
	for {
		if post.ParentID == nil {
			break
		}

		post, err = app.storage.FindPost(nil, clientID, nil, post.GroupID, *post.ParentID, true, false)
		if err != nil {
			log.Printf("error app.getPostToMemberList() - %s", err)
			return nil, fmt.Errorf("error app.getPostToMemberList() - %s", err)
		}

		if post != nil && len(post.ToMembersList) > 0 {
			return post.GetMembersAsUserIDs(skipUserID), nil
		}
	}

	return nil, nil
}

func (app *Application) updatePost(clientID string, current *model.User, group *model.Group, post *model.Post) (*model.Post, error) {
	if group.Settings != nil && !group.Settings.PostPreferences.CanSendPostToAdmins {
		userIDs := post.GetMembersAsUserIDs(&current.ID)
		memberships, err := app.Services.FindGroupMemberships(clientID, model.MembershipFilter{
			GroupIDs: []string{post.GroupID},
			UserIDs:  userIDs,
			Statuses: []string{"admin"},
		})
		if err != nil {
			return nil, err
		}

		if len(memberships.Items) > 0 {
			var toMembers []model.ToMember
			for _, membership := range memberships.Items {
				toMembers = append(toMembers, model.ToMember{
					UserID: membership.UserID,
					Name:   membership.Name,
				})
			}
			post.ToMembersList = toMembers
		}
	}

	return app.storage.UpdatePost(clientID, current.ID, post)
}

func (app *Application) reactToPost(clientID string, current *model.User, groupID string, postID string, reaction string) error {
	transaction := func(context storage.TransactionContext) error {
		post, err := app.storage.FindPost(context, clientID, &current.ID, groupID, postID, true, false)
		if err != nil {
			return fmt.Errorf("error finding post: %v", err)
		}
		if post == nil {
			return fmt.Errorf("missing post for id %s", postID)
		}

		for _, userID := range post.Reactions[reaction] {
			if current.ID == userID {
				err = app.storage.ReactToPost(context, current.ID, postID, reaction, false)
				if err != nil {
					return fmt.Errorf("error removing reaction: %v", err)
				}

				return nil
			}
		}

		err = app.storage.ReactToPost(context, current.ID, postID, reaction, true)
		if err != nil {
			return fmt.Errorf("error adding reaction: %v", err)
		}

		return nil
	}

	return app.storage.PerformTransaction(transaction)
}

func (app *Application) reportPostAsAbuse(clientID string, current *model.User, group *model.Group, post *model.Post, comment string, sendToDean bool, sendToGroupAdmins bool) error {

	if !sendToDean && !sendToGroupAdmins {
		sendToDean = true
	}

	err := app.storage.ReportPostAsAbuse(clientID, current.ID, group, post)
	if err != nil {
		log.Printf("error while reporting an abuse post: %s", err)
		return fmt.Errorf("error while reporting an abuse post: %s", err)
	}

	subject := ""
	if sendToDean && !sendToGroupAdmins {
		subject = "Report violation of Student Code to Dean of Students"
	} else if !sendToDean && sendToGroupAdmins {
		subject = "Report of Obscene, Harassing, or Threatening Content to Group Administrators"
	} else {
		subject = "Report violation of Student Code to Dean of Students and obscene, threatening, or harassing content to Group Administrators"
	}

	subject = fmt.Sprintf("%s %s", subject, post.DateCreated.Format(time.RFC850))

	if sendToDean {
		body := fmt.Sprintf(`
<div>Violation by: %s %s\n</div>
<div>Group title: %s\n</div>
<div>Post Title: %s\n</div>
<div>Post Body: %s\n</div>
<div>Reported by: %s %s\n</div>
<div>Reported comment: %s\n</div>
	`, current.ExternalID, post.Creator.Name, group.Title, post.Subject, post.Body,
			current.ExternalID, current.Name, comment)
		body = strings.ReplaceAll(body, `\n`, "\n")
		app.notifications.SendMail(app.config.ReportAbuseRecipientEmail, subject, body)
	}
	if sendToGroupAdmins {
		result, _ := app.storage.FindGroupMemberships(clientID, model.MembershipFilter{
			GroupIDs: []string{group.ID},
			Statuses: []string{"admin"},
		})
		toMembers := result.GetMembersAsRecipients(func(membership model.GroupMembership) (bool, bool) {
			return membership.UserID != current.ID, false
		})

		body := fmt.Sprintf(`
Violation by: %s %s
Group title: %s
Post Title: %s
Post Body: %s
Reported by: %s %s
Reported comment: %s
	`, current.ExternalID, post.Creator.Name, group.Title, post.Subject, post.Body,
			current.ExternalID, current.Name, comment)

		app.notifications.SendNotification(toMembers, nil, subject, body, map[string]string{
			"type":         "group",
			"operation":    "report_abuse_post",
			"entity_type":  "group",
			"entity_id":    group.ID,
			"entity_name":  group.Title,
			"post_id":      post.ID,
			"post_subject": post.Subject,
			"post_body":    post.Body,
		},
			current.AppID,
			current.OrgID)
	}

	return nil
}

func (app *Application) deletePost(clientID string, userID string, groupID string, postID string, force bool) error {
	return app.storage.DeletePost(nil, clientID, userID, groupID, postID, force)
}

func (app *Application) sendGroupNotification(clientID string, notification model.GroupNotification) error {
	memberStatuses := notification.MemberStatuses
	if len(memberStatuses) == 0 {
		memberStatuses = []string{"admin", "member"}
	}

	members, err := app.findGroupMemberships(clientID, model.MembershipFilter{
		GroupIDs: []string{notification.GroupID},
		UserIDs:  notification.Members.ToUserIDs(),
		Statuses: memberStatuses,
	})

	if err != nil {
		return err
	}

	app.sendNotification(members.GetMembersAsNotificationRecipients(func(member model.GroupMembership) (bool, bool) {
		return true, true // Should it be a separate notification preference?
	}), notification.Topic, notification.Subject, notification.Body, notification.Data, app.config.AppID, app.config.OrgID)

	return nil
}

func (app *Application) sendNotification(recipients []notifications.Recipient, topic *string, title string, text string, data map[string]string, appID string, orgID string) {
	app.notifications.SendNotification(recipients, topic, title, text, data, appID, orgID)
}

func (app *Application) getManagedGroupConfigs(clientID string) ([]model.ManagedGroupConfig, error) {
	return app.storage.FindManagedGroupConfigs(clientID)
}

func (app *Application) createManagedGroupConfig(config model.ManagedGroupConfig) (*model.ManagedGroupConfig, error) {
	config.ID = uuid.NewString()
	config.DateCreated = time.Now()
	config.DateUpdated = nil
	err := app.storage.InsertManagedGroupConfig(config)
	return &config, err
}

func (app *Application) updateManagedGroupConfig(config model.ManagedGroupConfig) error {
	return app.storage.UpdateManagedGroupConfig(config)
}

func (app *Application) deleteManagedGroupConfig(id string, clientID string) error {
	return app.storage.DeleteManagedGroupConfig(id, clientID)
}

func (app *Application) getSyncConfig(clientID string) (*model.SyncConfig, error) {
	return app.storage.FindSyncConfig(clientID)
}

func (app *Application) updateSyncConfig(config model.SyncConfig) error {
	return app.storage.SaveSyncConfig(nil, config)
}

func (app *Application) findGroupMembership(clientID string, groupID string, userID string) (*model.GroupMembership, error) {
	return app.storage.FindGroupMembership(clientID, groupID, userID)
}

func (app *Application) getResearchProfileUserCount(clientID string, current *model.User, researchProfile map[string]map[string][]string) (int64, error) {
	searchParams := app.formatCoreAccountSearchParams(researchProfile)
	return app.corebb.GetAccountsCount(searchParams, &current.AppID, &current.OrgID)
}

func (app *Application) formatCoreAccountSearchParams(researchProfile map[string]map[string][]string) map[string]interface{} {
	searchParams := map[string]interface{}{}
	for k1, v1 := range researchProfile {
		for k2, v2 := range v1 {
			searchParams["profile.unstructured_properties.research_questionnaire_answers."+k1+"."+k2] = map[string]interface{}{"operation": "any", "value": v2}
		}
	}
	// If empty profile is provided, find all users that have filled out the profile
	//TODO: Handle filled out profile search better
	if len(searchParams) == 0 {
		searchParams["profile.unstructured_properties.research_questionnaire_answers.demographics"] = "$exists"
	}

	return searchParams
}
