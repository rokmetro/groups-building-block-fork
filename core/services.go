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
	"errors"
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
	group, err := app.storage.FindGroup(clientID, id)
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
	if membership == nil || !membership.Admin {
		return false, nil
	}

	return true, nil
}

func (app *Application) getGroupStats(clientID string, id string) (*model.GroupStats, error) {
	return app.storage.GetGroupStats(clientID, id)
}

func (app *Application) createGroup(clientID string, current *model.User, group *model.Group) (*string, *utils.GroupError) {
	insertedID, err := app.storage.CreateGroup(clientID, current, group)
	if err != nil {
		return nil, err
	}

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

	return insertedID, nil
}

func (app *Application) updateGroup(clientID string, current *model.User, group *model.Group) *utils.GroupError {

	err := app.storage.UpdateGroupWithoutMembers(clientID, current, group)
	if err != nil {
		return err
	}
	return nil
}

func (app *Application) deleteGroup(clientID string, current *model.User, id string) error {
	err := app.storage.DeleteGroup(clientID, id)
	if err != nil {
		return err
	}
	return nil
}

func (app *Application) getGroups(clientID string, current *model.User, category *string, privacy *string, title *string, offset *int64, limit *int64, order *string) ([]model.Group, error) {

	// find the groups objects
	groups, err := app.storage.FindGroups(clientID, &current.ID, category, privacy, title, offset, limit, order)
	if err != nil {
		return nil, err
	}

	return groups, nil
}

func (app *Application) getAllGroups(clientID string) ([]model.Group, error) {
	// find the groups objects
	groups, err := app.storage.FindGroups(clientID, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	return groups, nil
}

func (app *Application) getUserGroups(clientID string, current *model.User, category *string, privacy *string, title *string, offset *int64, limit *int64, order *string) ([]model.Group, error) {
	// find the user groups
	groups, err := app.storage.FindUserGroups(clientID, current.ID, category, privacy, title, offset, limit, order)
	if err != nil {
		return nil, err
	}

	return groups, nil
}

func (app *Application) loginUser(clientID string, current *model.User) error {
	return app.storage.LoginUser(clientID, current)
}

func (app *Application) deleteUser(clientID string, current *model.User) error {
	return app.storage.DeleteUser(clientID, current.ID)
}

func (app *Application) getGroup(clientID string, current *model.User, id string) (*model.Group, error) {
	// find the group
	group, err := app.storage.FindGroup(clientID, id)
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (app *Application) createPendingMember(clientID string, current *model.User, group *model.Group, member *model.Member) error {

	if group.CanJoinAutomatically {
		member.Status = "member"
	} else {
		member.Status = "pending"
	}

	err := app.storage.CreatePendingMember(clientID, current, group, member)
	if err != nil {
		return err
	}

	adminMemberships, err := app.storage.FindGroupMemberships(clientID, model.MembershipFilter{
		GroupIDs: []string{group.ID},
		Statuses: []string{"admin"},
	})
	if err == nil && len(adminMemberships.Items) > 0 {
		if len(adminMemberships.Items) > 0 {
			recipients := []notifications.Recipient{}
			for _, admin := range adminMemberships.Items {
				recipients = append(recipients, notifications.Recipient{
					UserID: admin.UserID,
					Name:   admin.Name,
				})
			}
			if len(recipients) > 0 {
				topic := "group.invitations"

				message := fmt.Sprintf("New membership request for '%s' group has been submitted", group.Title)
				if group.CanJoinAutomatically {
					message = fmt.Sprintf("%s joined '%s' group", member.GetDisplayName(), group.Title)
				}

				app.notifications.SendNotification(
					recipients,
					&topic,
					fmt.Sprintf("Group - %s", group.Title),
					message,
					map[string]string{
						"type":        "group",
						"operation":   "pending_member",
						"entity_type": "group",
						"entity_id":   group.ID,
						"entity_name": group.Title,
					},
				)
			}
		}
	} else {
		log.Printf("Unable to retrieve group by membership id: %s\n", err)
		// return err // No reason to fail if the main part succeeds
	}

	if group.CanJoinAutomatically && group.AuthmanEnabled {
		err := app.authman.AddAuthmanMemberToGroup(*group.AuthmanGroup, member.ExternalID)
		if err != nil {
			log.Printf("err app.createPendingMember() - error storing member in Authman: %s", err)
		}
	}

	return nil
}

func (app *Application) deletePendingMember(clientID string, current *model.User, groupID string) error {
	err := app.storage.DeletePendingMember(clientID, groupID, current.ID)
	if err != nil {
		return err
	}

	group, err := app.storage.FindGroup(clientID, groupID)
	if err == nil && group != nil {
		if group.CanJoinAutomatically && group.AuthmanEnabled {
			err := app.authman.RemoveAuthmanMemberFromGroup(*group.AuthmanGroup, current.ExternalID)
			if err != nil {
				log.Printf("err app.createPendingMember() - error storing member in Authman: %s", err)
			}
		}
	}

	return nil
}

func (app *Application) createMember(clientID string, current *model.User, group *model.Group, member *model.Member) error {

	if (member.UserID == "" && member.ExternalID != "") ||
		(member.UserID != "" && member.ExternalID == "") {
		if member.ExternalID == "" {
			user, err := app.storage.FindUser(clientID, member.UserID, false)
			if err == nil && user != nil {
				member.ApplyFromUserIfEmpty(user)
			} else {
				log.Printf("error app.createMember() - unable to find user: %s", err)
			}
		}
		if member.UserID == "" {
			user, err := app.storage.FindUser(clientID, member.ExternalID, true)
			if err == nil && user != nil {
				member.ApplyFromUserIfEmpty(user)
			} else {
				log.Printf("error app.createMember() - unable to find user: %s", err)
			}
		}
	}

	err := app.storage.CreateMemberUnchecked(clientID, current, group, member)
	if err != nil {
		return err
	}

	memberships, err := app.storage.FindGroupMemberships(clientID, model.MembershipFilter{
		GroupIDs: []string{group.ID},
		Statuses: []string{"admin"},
	})
	if err == nil && len(memberships.Items) > 0 {
		recipients := []notifications.Recipient{}
		for _, adminMember := range memberships.Items {
			if adminMember.UserID != current.ID {
				recipients = append(recipients, notifications.Recipient{
					UserID: adminMember.UserID,
					Name:   adminMember.Name,
				})
			}
		}

		var message string
		if member.Status == "member" || member.Status == "admin" {
			message = fmt.Sprintf("New member joined '%s' group", group.Title)
		} else {
			message = fmt.Sprintf("New membership request for '%s' group has been submitted", group.Title)
		}

		if len(recipients) > 0 {
			topic := "group.invitations"
			app.notifications.SendNotification(
				recipients,
				&topic,
				fmt.Sprintf("Group - %s", group.Title),
				message,
				map[string]string{
					"type":        "group",
					"operation":   "pending_member",
					"entity_type": "group",
					"entity_id":   group.ID,
					"entity_name": group.Title,
				},
			)

		}

		if group.AuthmanEnabled && group.AuthmanGroup != nil {
			err = app.authman.AddAuthmanMemberToGroup(*group.AuthmanGroup, member.ExternalID)
			if err != nil {
				return err
			}
		}

	} else if err != nil {
		log.Printf("Unable to retrieve group by membership id: %s\n", err)
		// return err // No reason to fail if the main part succeeds
	}
	if err == nil && group != nil {
		if group.CanJoinAutomatically && group.AuthmanEnabled {
			err := app.authman.AddAuthmanMemberToGroup(*group.AuthmanGroup, current.ExternalID)
			if err != nil {
				log.Printf("err app.createMember() - error storing member in Authman: %s", err)
			}
		}
	}

	return nil
}

func (app *Application) deleteMember(clientID string, current *model.User, groupID string) error {
	err := app.storage.DeleteMember(clientID, groupID, current.ID, false)
	if err != nil {
		return err
	}

	group, err := app.storage.FindGroup(clientID, groupID)
	if err == nil && group != nil {
		if group.CanJoinAutomatically && group.AuthmanEnabled {
			err := app.authman.RemoveAuthmanMemberFromGroup(*group.AuthmanGroup, current.ExternalID)
			if err != nil {
				log.Printf("err app.createPendingMember() - error storing member in Authman: %s", err)
			}
		}
	}

	return nil
}

func (app *Application) applyMembershipApproval(clientID string, current *model.User, membershipID string, approve bool, rejectReason string) error {
	err := app.storage.ApplyMembershipApproval(clientID, membershipID, approve, rejectReason)
	if err != nil {
		return fmt.Errorf("error applying membership approval: %s", err)
	}

	membership, err := app.storage.FindGroupMembershipByID(clientID, membershipID)
	if err == nil && membership != nil {
		group, _ := app.storage.FindGroup(clientID, membership.GroupID)
		topic := "group.invitations"
		if approve {
			app.notifications.SendNotification(
				[]notifications.Recipient{
					notifications.Recipient{
						UserID: membership.UserID,
						Name:   membership.Name,
					},
				},
				&topic,
				fmt.Sprintf("Group - %s", group.Title),
				fmt.Sprintf("Your membership in '%s' group has been approved", group.Title),
				map[string]string{
					"type":        "group",
					"operation":   "membership_approve",
					"entity_type": "group",
					"entity_id":   group.ID,
					"entity_name": group.Title,
				},
			)
		} else {
			app.notifications.SendNotification(
				[]notifications.Recipient{
					notifications.Recipient{
						UserID: membership.UserID,
						Name:   membership.Name,
					},
				},
				&topic,
				fmt.Sprintf("Group - %s", group.Title),
				fmt.Sprintf("Your membership in '%s' group has been rejected with a reason: %s", group.Title, rejectReason),
				map[string]string{
					"type":        "group",
					"operation":   "membership_reject",
					"entity_type": "group",
					"entity_id":   group.ID,
					"entity_name": group.Title,
				},
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

func (app *Application) deleteMembership(clientID string, current *model.User, membershipID string) error {
	err := app.storage.DeleteMembership(clientID, current, membershipID)
	if err != nil {
		return err
	}

	membership, _ := app.storage.FindGroupMembershipByID(clientID, membershipID)
	if membership != nil {
		group, _ := app.storage.FindGroup(clientID, membership.GroupID)
		if group.CanJoinAutomatically && group.AuthmanEnabled && membership.ExternalID != "" {
			err := app.authman.RemoveAuthmanMemberFromGroup(*group.AuthmanGroup, membership.ExternalID)
			if err != nil {
				log.Printf("err app.createPendingMember() - error storing member in Authman: %s", err)
			}
		}
	}
	return nil
}

func (app *Application) updateMembership(clientID string, current *model.User, membershipID string, status string, dateAttended *time.Time) error {
	err := app.storage.UpdateMembership(clientID, current, membershipID, status, dateAttended)
	if err != nil {
		return err
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

	var recipients []notifications.Recipient
	if len(event.ToMembersList) > 0 {
		recipients = event.GetMembersAsNotificationRecipients(skipUserID)
	} else {
		result, _ := app.storage.FindGroupMemberships(clientID, model.MembershipFilter{
			GroupIDs: []string{group.ID},
			Statuses: []string{"member", "admin"},
		})
		recipients = result.GetMembersAsNotificationRecipients(skipUserID)
	}
	topic := "group.events"
	app.notifications.SendNotification(
		recipients,
		&topic,
		fmt.Sprintf("Group - %s", group.Title),
		fmt.Sprintf("New event has been published in '%s' group", group.Title),
		map[string]string{
			"type":        "group",
			"operation":   "event_created",
			"entity_type": "group",
			"entity_id":   group.ID,
			"entity_name": group.Title,
		},
	)

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

func (app *Application) getPosts(clientID string, current *model.User, groupID string, filterPrivatePostsValue *bool, filterByToMembers bool, offset *int64, limit *int64, order *string) ([]*model.Post, error) {
	return app.storage.FindPosts(clientID, current, groupID, filterPrivatePostsValue, filterByToMembers, offset, limit, order)
}

func (app *Application) getPost(clientID string, userID *string, groupID string, postID string, skipMembershipCheck bool, filterByToMembers bool) (*model.Post, error) {
	return app.storage.FindPost(clientID, userID, groupID, postID, skipMembershipCheck, filterByToMembers)
}

func (app *Application) getUserPostCount(clientID string, userID string) (*int64, error) {
	return app.storage.GetUserPostCount(clientID, userID)
}

func (app *Application) createPost(clientID string, current *model.User, post *model.Post, group *model.Group) (*model.Post, error) {
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

		recipients, _ := app.getPostNotificationRecipients(clientID, post, &current.ID)

		if len(recipients) == 0 {
			result, _ := app.storage.FindGroupMemberships(clientID, model.MembershipFilter{
				GroupIDs: []string{group.ID},
				Statuses: []string{"member", "admin"},
			})
			recipients = result.GetMembersAsNotificationRecipients(&current.ID)
		}
		if len(recipients) > 0 {
			title := fmt.Sprintf("Group - %s", group.Title)
			body := fmt.Sprintf("New post has been published in '%s' group", group.Title)
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
					"post_id":      *post.ID,
					"post_subject": post.Subject,
					"post_body":    post.Body,
				},
			)
		}
	}
	go handleNotification()

	return post, nil
}

func (app *Application) getPostNotificationRecipients(clientID string, post *model.Post, skipUserID *string) ([]notifications.Recipient, error) {
	if post == nil {
		return nil, nil
	}

	if len(post.ToMembersList) > 0 {
		return post.GetMembersAsNotificationRecipients(skipUserID), nil
	}

	var err error
	for {
		if post.ParentID == nil {
			break
		}

		post, err = app.storage.FindPost(clientID, nil, post.GroupID, *post.ParentID, true, false)
		if err != nil {
			log.Printf("error app.getPostToMemberList() - %s", err)
			return nil, fmt.Errorf("error app.getPostToMemberList() - %s", err)
		}

		if post != nil && len(post.ToMembersList) > 0 {
			return post.GetMembersAsNotificationRecipients(skipUserID), nil
		}
	}

	return nil, nil
}

func (app *Application) updatePost(clientID string, current *model.User, post *model.Post) (*model.Post, error) {
	return app.storage.UpdatePost(clientID, current.ID, post)
}

func (app *Application) reportPostAsAbuse(clientID string, current *model.User, group *model.Group, post *model.Post, comment string, sendToDean bool, sendToGroupAdmins bool) error {

	if !sendToDean && !sendToGroupAdmins {
		sendToDean = true
	}

	var creatorExternalID string
	creator, err := app.storage.FindUser(clientID, post.Creator.UserID, false)
	if err != nil {
		log.Printf("error retrieving user: %s", err)
	} else if creator != nil {
		creatorExternalID = creator.ExternalID
	}

	err = app.storage.ReportPostAsAbuse(clientID, current.ID, group, post)
	if err != nil {
		log.Printf("error while reporting an abuse post: %s", err)
		return fmt.Errorf("error while reporting an abuse post: %s", err)
	}

	subject := ""
	if sendToDean && !sendToGroupAdmins {
		subject = "Report violation of Student Code to Dean of Students"
	} else if !sendToDean && sendToGroupAdmins {
		subject = "Report obscene, threatening, or harassing content to Group Administrators"
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
	`, creatorExternalID, post.Creator.Name, group.Title, post.Subject, post.Body,
			current.ExternalID, current.Name, comment)
		body = strings.ReplaceAll(body, `\n`, "\n")
		app.notifications.SendMail(app.config.ReportAbuseRecipientEmail, subject, body)
	}
	if sendToGroupAdmins {
		result, _ := app.storage.FindGroupMemberships(clientID, model.MembershipFilter{
			GroupIDs: []string{group.ID},
			Statuses: []string{"admin"},
		})
		toMembers := result.GetMembersAsRecipients(nil)

		body := fmt.Sprintf(`
Violation by: %s %s
Group title: %s
Post Title: %s
Post Body: %s
Reported by: %s %s
Reported comment: %s
	`, creatorExternalID, post.Creator.Name, group.Title, post.Subject, post.Body,
			current.ExternalID, current.Name, comment)

		app.notifications.SendNotification(toMembers, nil, subject, body, map[string]string{
			"type":         "group",
			"operation":    "report_abuse_post",
			"entity_type":  "group",
			"entity_id":    group.ID,
			"entity_name":  group.Title,
			"post_id":      *post.ID,
			"post_subject": post.Subject,
			"post_body":    post.Body,
		})
	}

	return nil
}

func (app *Application) deletePost(clientID string, userID string, groupID string, postID string, force bool) error {
	return app.storage.DeletePost(clientID, userID, groupID, postID, force)
}

// TODO this logic needs to be refactored because it's over complicated!
func (app *Application) synchronizeAuthman(clientID string, checkThreshold bool) error {
	startTime := time.Now()
	transaction := func(context storage.TransactionContext) error {
		times, err := app.storage.FindSyncTimes(context, clientID)
		if err != nil {
			return err
		}
		if times != nil && times.StartTime != nil {
			config, err := app.storage.FindSyncConfig(clientID)
			if err != nil {
				log.Printf("error finding sync configs for clientID %s: %v", clientID, err)
			}
			timeout := defaultConfigSyncTimeout
			if config != nil && config.Timeout > 0 {
				timeout = config.Timeout
			}
			if times.EndTime == nil {
				if !startTime.After(times.StartTime.Add(time.Minute * time.Duration(timeout))) {
					log.Println("Another Authman sync process is running for clientID " + clientID)
					return fmt.Errorf("another Authman sync process is running" + clientID)
				}
				log.Printf("Authman sync past timeout threshold %d mins for client ID %s\n", timeout, clientID)
			}
			if checkThreshold {
				if config == nil {
					log.Printf("missing sync configs for clientID %s", clientID)
					return fmt.Errorf("missing sync configs for clientID %s: %v", clientID, err)
				}
				if !startTime.After(times.StartTime.Add(time.Minute * time.Duration(config.TimeThreshold))) {
					log.Println("Authman has already been synced for clientID " + clientID)
					return fmt.Errorf("Authman has already been synced for clientID %s", clientID)
				}
			}
		}

		return app.storage.SaveSyncTimes(context, model.SyncTimes{StartTime: &startTime, EndTime: nil, ClientID: clientID})
	}

	err := app.storage.PerformTransaction(transaction)
	if err != nil {
		return err
	}

	log.Printf("Global Authman synchronization started for clientID: %s\n", clientID)

	app.authmanSyncInProgress = true
	finishAuthmanSync := func() {
		endTime := time.Now()
		err := app.storage.SaveSyncTimes(nil, model.SyncTimes{StartTime: &startTime, EndTime: &endTime, ClientID: clientID})
		if err != nil {
			log.Printf("Error saving sync configs to end sync: %s\n", err)
			return
		}
		log.Printf("Global Authman synchronization finished for clientID: %s\n", clientID)
	}
	defer finishAuthmanSync()

	configs, err := app.storage.FindManagedGroupConfigs(clientID)
	if err != nil {
		return fmt.Errorf("error finding managed group configs for clientID %s", clientID)
	}

	for _, config := range configs {
		for _, stemName := range config.AuthmanStems {
			stemGroups, err := app.authman.RetrieveAuthmanStemGroups(stemName)
			if err != nil {
				return fmt.Errorf("error on requesting Authman for stem groups: %s", err)
			}

			if stemGroups != nil && len(stemGroups.WsFindGroupsResults.GroupResults) > 0 {
				for _, stemGroup := range stemGroups.WsFindGroupsResults.GroupResults {
					storedStemGroup, err := app.storage.FindAuthmanGroupByKey(clientID, stemGroup.Name)
					if err != nil {
						return fmt.Errorf("error on requesting Authman for stem groups: %s", err)
					}

					title, adminUINs := stemGroup.GetGroupPrettyTitleAndAdmins()

					defaultAdminsMapping := map[string]bool{}
					for _, externalID := range adminUINs {
						defaultAdminsMapping[externalID] = true
					}
					for _, externalID := range app.config.AuthmanAdminUINList {
						defaultAdminsMapping[externalID] = true
					}
					for _, externalID := range config.AdminUINs {
						defaultAdminsMapping[externalID] = true
					}

					constructedAdminUINs := []string{}
					if len(defaultAdminsMapping) > 0 {
						for key := range defaultAdminsMapping {
							constructedAdminUINs = append(constructedAdminUINs, key)
						}
					}

					if storedStemGroup == nil {
						var members []model.Member
						if len(constructedAdminUINs) > 0 {
							members = app.buildMembersByExternalIDs(clientID, constructedAdminUINs, "admin")
						}

						emptyText := ""
						_, err := app.storage.CreateGroup(clientID, nil, &model.Group{
							Title:                title,
							Description:          &emptyText,
							Category:             "Academic", // Hardcoded.
							Privacy:              "private",
							HiddenForSearch:      true,
							CanJoinAutomatically: true,
							AuthmanEnabled:       true,
							AuthmanGroup:         &stemGroup.Name,
							Members:              members,
						})
						if err != nil {
							return fmt.Errorf("error on create Authman stem group: '%s' - %s", stemGroup.Name, err)
						}

						log.Printf("Created new `%s` group", title)
					} else {
						missedUINs := []string{}
						groupUpdated := false
						for _, uin := range adminUINs {
							found := false
							for index, member := range storedStemGroup.Members {
								if member.ExternalID == uin {
									if member.Status != "admin" {
										now := time.Now()
										storedStemGroup.Members[index].Status = "admin"
										storedStemGroup.Members[index].DateUpdated = &now
										groupUpdated = true
										break
									}
									found = true
								}
							}
							if !found {
								missedUINs = append(missedUINs, uin)
							}
						}

						if len(missedUINs) > 0 {
							missedMembers := app.buildMembersByExternalIDs(clientID, missedUINs, "admin")
							if len(missedMembers) > 0 {
								storedStemGroup.Members = append(storedStemGroup.Members, missedMembers...)
								groupUpdated = true
							}
						}

						if storedStemGroup.Title != title {
							storedStemGroup.Title = title
							groupUpdated = true
						}

						if storedStemGroup.Category == "" {
							storedStemGroup.Category = "Academic" // Hardcoded.
							groupUpdated = true
						}

						if groupUpdated {
							err := app.storage.UpdateGroupWithMembers(clientID, nil, storedStemGroup)
							if err != nil {
								fmt.Errorf("error app.synchronizeAuthmanGroup() - unable to update group admins of '%s' - %s", storedStemGroup.Title, err)
							}
						}
					}
				}
			}
		}
	}

	authmanGroups, err := app.storage.FindAuthmanGroups(clientID)
	if err != nil {
		return err
	}

	if len(authmanGroups) > 0 {
		for _, authmanGroup := range authmanGroups {
			err := app.synchronizeAuthmanGroup(clientID, authmanGroup.ID)
			if err != nil {
				fmt.Errorf("error app.synchronizeAuthmanGroup() '%s' - %s", authmanGroup.Title, err)
			}
		}
	}

	return nil
}

func (app *Application) buildMembersByExternalIDs(clientID string, externalIDs []string, memberStatus string) []model.Member {
	if len(externalIDs) > 0 {
		users, _ := app.storage.FindUsers(clientID, externalIDs, true)
		members := []model.Member{}
		userExternalIDmapping := map[string]model.User{}
		for _, user := range users {
			userExternalIDmapping[user.ExternalID] = user
		}

		for _, externalID := range externalIDs {
			if value, ok := userExternalIDmapping[externalID]; ok {
				members = append(members, model.Member{
					ID:          uuid.NewString(),
					UserID:      value.ID,
					ExternalID:  externalID,
					Name:        value.Name,
					Email:       value.Email,
					Status:      memberStatus,
					DateCreated: time.Now(),
				})
			} else {
				members = append(members, model.Member{
					ID:          uuid.NewString(),
					ExternalID:  externalID,
					Status:      memberStatus,
					DateCreated: time.Now(),
				})
			}
		}
		return members
	}
	return nil
}

// TODO this logic needs to be refactored because it's over complicated!
func (app *Application) synchronizeAuthmanGroup(clientID string, groupID string) error {
	if groupID == "" {
		return errors.New("Missing group ID")
	}
	var group *model.Group
	var err error
	group, err = app.checkGroupSyncTimes(clientID, groupID)
	if err != nil {
		return err
	}

	log.Printf("Authman synchronization for group %s started", *group.AuthmanGroup)

	authmanExternalIDs, authmanErr := app.authman.RetrieveAuthmanGroupMembers(*group.AuthmanGroup)
	if authmanErr != nil {
		return fmt.Errorf("error on requesting Authman for %s: %s", *group.AuthmanGroup, authmanErr)
	}

	app.authmanSyncInProgress = true
	finishAuthmanSync := func() {
		endTime := time.Now()
		group.SyncEndTime = &endTime
		err = app.storage.UpdateGroupSyncTimes(nil, clientID, group)
		if err != nil {
			log.Printf("Error saving group to end sync for Authman %s: %s\n", *group.AuthmanGroup, err)
			return
		}
		log.Printf("Authman synchronization for group %s finished", *group.AuthmanGroup)
	}
	defer finishAuthmanSync()

	err = app.syncAuthmanGroupMemberships(clientID, group, authmanExternalIDs)
	if err != nil {
		return fmt.Errorf("error updating group memberships for Authman %s: %s", *group.AuthmanGroup, err)
	}

	return nil
}

func (app *Application) checkGroupSyncTimes(clientID string, groupID string) (*model.Group, error) {
	var group *model.Group
	var err error
	startTime := time.Now()
	transaction := func(context storage.TransactionContext) error {
		group, err = app.storage.FindGroupWithContext(context, clientID, groupID)
		if err != nil {
			return fmt.Errorf("error finding group for ID %s: %s", groupID, err)
		}
		if group == nil {
			return fmt.Errorf("missing group for ID %s", groupID)
		}
		if !group.IsAuthmanSyncEligible() {
			return fmt.Errorf("Authman synchronization failed for group '%s' due to bad settings", group.Title)
		}

		if group.SyncStartTime != nil {
			config, err := app.storage.FindSyncConfig(clientID)
			if err != nil {
				log.Printf("error finding sync configs for clientID %s: %v", clientID, err)
			}
			timeout := defaultConfigSyncTimeout
			if config != nil && config.GroupTimeout > 0 {
				timeout = config.GroupTimeout
			}
			if group.SyncEndTime == nil {
				if !startTime.After(group.SyncStartTime.Add(time.Minute * time.Duration(timeout))) {
					log.Println("Another Authman sync process is running for group ID " + group.ID)
					return fmt.Errorf("another Authman sync process is running for group ID %s", group.ID)
				}
				log.Printf("Authman sync timed out after %d mins for group ID %s\n", timeout, group.ID)
			}
		}

		group.SyncStartTime = &startTime
		group.SyncEndTime = nil
		err = app.storage.UpdateGroupSyncTimes(context, clientID, group)
		if err != nil {
			return fmt.Errorf("error switching to group memberships for Authman %s: %s", *group.AuthmanGroup, err)
		}
		return nil
	}

	err = app.storage.PerformTransaction(transaction)
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (app *Application) syncAuthmanGroupMemberships(clientID string, authmanGroup *model.Group, authmanExternalIDs []string) error {
	syncID := uuid.NewString()
	log.Printf("Sync ID %s for Authman %s...\n", syncID, *authmanGroup.AuthmanGroup)

	// Get list of all member external IDs (Authman members + admins)
	allExternalIDs := append([]string{}, authmanExternalIDs...)
	adminMembers, err := app.storage.FindGroupMemberships(clientID, model.MembershipFilter{
		GroupIDs: []string{authmanGroup.ID},
		Statuses: []string{"admin"},
	})
	if err != nil {
		log.Printf("Error finding admin memberships in Authman %s: %s\n", *authmanGroup.AuthmanGroup, err)
	} else {
		for _, adminMember := range adminMembers.Items {
			if len(adminMember.ExternalID) > 0 {
				allExternalIDs = append(allExternalIDs, adminMember.ExternalID)
			}
		}
	}

	// Load user records for all members
	localUsersMapping := map[string]model.User{}
	localUsers, err := app.storage.FindUsers(clientID, allExternalIDs, true)
	if err != nil {
		return fmt.Errorf("error on getting %d users for Authman %s: %s", len(allExternalIDs), *authmanGroup.AuthmanGroup, err)
	}

	for _, user := range localUsers {
		localUsersMapping[user.ExternalID] = user
	}

	missingInfoMembers := []model.GroupMembership{}

	//TODO: Move this migration to handle all groups (not just authman)
	// Transfer existing embedded group members
	log.Printf("Transferring %d existing embedded members for Authman %s...\n", len(authmanGroup.Members), *authmanGroup.AuthmanGroup)
	for _, member := range authmanGroup.Members {
		membership := member.ToGroupMembership(clientID, authmanGroup.ID)
		err := app.storage.CreateMissingGroupMembership(&membership)
		if err != nil {
			log.Printf("Error transferring embedded membership for external ID %s in Authman %s: %s\n", member.ExternalID, *authmanGroup.AuthmanGroup, err)
		}
	}

	log.Printf("Processing %d current members for Authman %s...\n", len(authmanExternalIDs), *authmanGroup.AuthmanGroup)
	for _, externalID := range authmanExternalIDs {
		status := "member"
		var userID *string
		var name *string
		var email *string
		if user, ok := localUsersMapping[externalID]; ok {
			if user.ID != "" {
				userID = &user.ID
			}
			if user.Name != "" {
				name = &user.Name
			}
			if user.Email != "" {
				email = &user.Email
			}
		}

		membership, err := app.storage.SaveGroupMembershipByExternalID(clientID, authmanGroup.ID, externalID, userID, &status, nil, email,
			name, authmanGroup.CreateMembershipEmptyAnswers(), &syncID)
		if err != nil {
			log.Printf("Error saving membership for external ID %s in Authman %s: %s\n", externalID, *authmanGroup.AuthmanGroup, err)
		} else if membership.Email == "" || membership.Name == "" {
			missingInfoMembers = append(missingInfoMembers, *membership)
		}
	}

	// Update admin user data
	for _, adminMember := range adminMembers.Items {
		var userID *string
		var name *string
		var email *string
		updatedInfo := false
		if mappedUser, ok := localUsersMapping[adminMember.ExternalID]; ok {
			if mappedUser.ID != "" && mappedUser.ID != adminMember.UserID {
				userID = &mappedUser.ID
				updatedInfo = true
			}
			if mappedUser.Name != "" && mappedUser.Name != adminMember.Name {
				name = &mappedUser.Name
				updatedInfo = true
			}
			if mappedUser.Email != "" && mappedUser.Email != adminMember.Email {
				email = &mappedUser.Email
				updatedInfo = true
			}
		}
		if updatedInfo {
			_, err := app.storage.SaveGroupMembershipByExternalID(clientID, authmanGroup.ID, adminMember.ExternalID, userID, nil, nil, email, name, nil, nil)
			if err != nil {
				log.Printf("Error saving admin membership with missing info for external ID %s in Authman %s: %s\n", adminMember.ExternalID, *authmanGroup.AuthmanGroup, err)
			}
		}
	}

	// Fetch user info for the required users
	log.Printf("Processing %d members missing info for Authman %s...\n", len(missingInfoMembers), *authmanGroup.AuthmanGroup)
	for i := 0; i < len(missingInfoMembers); i += authmanUserBatchSize {
		j := i + authmanUserBatchSize
		if j > len(missingInfoMembers) {
			j = len(missingInfoMembers)
		}
		log.Printf("Processing members missing info %d - %d for Authman %s...\n", i, j, *authmanGroup.AuthmanGroup)
		members := missingInfoMembers[i:j]
		externalIDs := make([]string, j-i)
		for i, member := range members {
			externalIDs[i] = member.ExternalID
		}
		authmanUsers, err := app.authman.RetrieveAuthmanUsers(externalIDs)
		if err != nil {
			log.Printf("error on retrieving missing user info for %d members: %s\n", len(externalIDs), err)
		} else if len(authmanUsers) > 0 {
			for _, member := range members {
				var name *string
				var email *string
				updatedInfo := false
				if mappedUser, ok := authmanUsers[member.ExternalID]; ok {
					if member.Name == "" && mappedUser.Name != "" {
						name = &mappedUser.Name
						updatedInfo = true
					}
					if member.Email == "" && len(mappedUser.AttributeValues) > 0 {
						email = &mappedUser.AttributeValues[0]
						updatedInfo = true
					}
					if !updatedInfo {
						log.Printf("The user has missing info: %+v Group: '%s' Authman Group: '%s'\n", mappedUser, authmanGroup.Title, *authmanGroup.AuthmanGroup)
					}
				}
				if updatedInfo {
					_, err := app.storage.SaveGroupMembershipByExternalID(clientID, authmanGroup.ID, member.ExternalID, nil, nil, nil, email, name, nil, nil)
					if err != nil {
						log.Printf("Error saving membership with missing info for external ID %s in Authman %s: %s\n", member.ExternalID, *authmanGroup.AuthmanGroup, err)
					}
				}
			}
		}
	}

	// Delete removed non-admin members
	log.Printf("Deleting removed members for Authman %s...\n", *authmanGroup.AuthmanGroup)
	admin := false
	deleteCount, err := app.storage.DeleteUnsyncedGroupMemberships(clientID, authmanGroup.ID, syncID, &admin)
	if err != nil {
		log.Printf("Error deleting removed memberships in Authman %s\n", *authmanGroup.AuthmanGroup)
	} else {
		log.Printf("%d memberships removed from Authman %s\n", deleteCount, *authmanGroup.AuthmanGroup)
	}

	return nil
}

func (app *Application) sendNotification(recipients []notifications.Recipient, topic *string, title string, text string, data map[string]string) {
	app.notifications.SendNotification(recipients, topic, title, text, data)
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
